package storage

import (
	"My-Redis/config"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

//const (
//	bufferSize    = 32 * 1024 // 32 KB
//	newLine       = '\n'
//	openBrace     = '{'
//	closeBrace    = '}'
//	minJSONLength = 2
//)

type RequestData struct {
	NameBD    *string        `json:"NameBD,omitempty"`    // можно опустить, если не обязателен
	NameTable *string        `json:"NameTable,omitempty"` // можно опустить, если не обязателен
	Data      map[string]any `json:"Data"`                // Без указателя ибо map по умолчанию указателен
	// остальные поля не декодируются и не занимают память
}

// [каталог][файл][таблицы]
// максимум (int16)32767
type memoryTablesType struct {
	storage   map[string]map[string]interface{}
	maxTables int16
	mu        sync.RWMutex
}

func NewMemoryTablesType(maxTables int16) memoryTablesType {
	return memoryTablesType{
		storage:   make(map[string]map[string]interface{}, 1),
		maxTables: maxTables,
	}
}

// Определяем структуру хранилища
type BaseStorage struct {
	config       config.Config
	memoryTables memoryTablesType
}

// Инициализация выбранного БД(только 1 раз)
func NewBaseStorage(cfg config.Config) *BaseStorage {
	return &BaseStorage{
		config:       cfg,
		memoryTables: NewMemoryTablesType(10), // Инициализация с рабочей map
	}
}

//func (a *a) UploadInColdStorage(NameBD, NameTable string, Data map[string]interface{}) error {
//	basePath := filepath.Join(a.Config.PathEXE, ".redis", NameBD)
//	if err := os.MkdirAll(basePath, 0755); err != nil {
//		return fmt.Errorf("failed to create directory: %w", err)
//	}
//
//	tablePath := filepath.Join(basePath, NameTable+".db")
//	file, err := os.OpenFile(tablePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
//	if err != nil {
//		return fmt.Errorf("failed to open file: %w", err)
//	}
//	defer file.Close()
//
//	bufferedWriter := bufio.NewWriterSize(file, bufferSize)
//	defer func() {
//		if flushErr := bufferedWriter.Flush(); flushErr != nil && err == nil {
//			err = fmt.Errorf("buffer flush failed: %w", flushErr)
//		}
//	}()
//
//	// Получаем буфер из пула
//	buf := a.bufferPool.Get().(*bytes.Buffer)
//	defer func() {
//		if buf.Cap() > 32*1024 { // Не возвращаем буферы >32 КБ
//			return
//		}
//		buf.Reset()
//		a.bufferPool.Put(buf)
//	}()
//
//	// Кодируем данные в буфер
//	enc := json.NewEncoder(buf)
//	if err := enc.Encode(Data); err != nil {
//		return fmt.Errorf("json encoding failed: %w", err)
//	}
//
//	// Получаем байты и проверяем длину
//	encodedData := buf.Bytes()
//	if len(encodedData) < minJSONLength+1 { // +1 для символа новой строки
//		return fmt.Errorf("invalid JSON format: too short")
//	}
//
//	// Убираем внешние скобки и символ новой строки
//	trimmedData := encodedData[1 : len(encodedData)-2]
//
//	// Проверка структуры
//	if encodedData[0] != openBrace || encodedData[len(encodedData)-2] != closeBrace {
//		return fmt.Errorf("invalid JSON format: missing braces")
//	}
//
//	println(string(trimmedData))
//	// Записываем данные
//	if _, err := bufferedWriter.Write(trimmedData); err != nil {
//		return fmt.Errorf("write to buffer failed: %w", err)
//	}
//	if err := bufferedWriter.WriteByte(newLine); err != nil {
//		return fmt.Errorf("failed to write newline: %w", err)
//	}
//
//	return nil
//}

func (*BaseStorage) DELETE(w http.ResponseWriter, r *http.Request) {}
func (a *BaseStorage) GET(w http.ResponseWriter, r *http.Request) {
	ok, data := a.rowIsExist(w, r, true)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	} else {
		json.NewEncoder(w).Encode(data)
		return
	}
}

func (a *BaseStorage) SET(w http.ResponseWriter, r *http.Request) {
	var data RequestData
	_ = processBodyToData(w, r, &data)

	if data.NameBD == nil || data.NameTable == nil {
		http.Error(w, `{"status":"error","message":"Missing required fields"}`, http.StatusBadRequest)
		return
	}
	_ = a.addInMemory(*data.NameBD, *data.NameTable, data.Data)
}

func (a *BaseStorage) IsExist(w http.ResponseWriter, r *http.Request) {
	ok, _ := a.rowIsExist(w, r, false)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	return
}
func processBodyToData(w http.ResponseWriter, r *http.Request, data *RequestData) bool {
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		http.Error(w, `{"status":"error","message":"Invalid JSON"}`, http.StatusBadRequest)
		return false
	}
	defer r.Body.Close()
	return true
}

func (a *BaseStorage) addInMemory(dbName, tableName string, row map[string]any) error {
	a.memoryTables.mu.Lock()
	defer a.memoryTables.mu.Unlock()

	// Инициализация базы данных, если её нет
	if a.memoryTables.storage[dbName] == nil {
		a.memoryTables.storage[dbName] = make(map[string]interface{})
	}

	// Инициализация таблицы, если её нет
	db := a.memoryTables.storage[dbName]
	if db[tableName] == nil {
		db[tableName] = make(map[string]map[string]any)
	}

	// Приводим таблицу к нужному типу
	tableData := db[tableName].(map[string]map[string]any)

	// Получаем ключ (более безопасный способ)
	if len(row) != 1 {
		return fmt.Errorf("row must contain exactly one key-value pair")
	}

	var key string
	for k := range row {
		key = k
		break
	}

	// Определяем операцию (добавление или изменение)
	operation := "added"
	if _, exists := tableData[key]; exists {
		operation = "updated"
	}

	// Проверяем и приводим тип значения
	value, ok := row[key].(map[string]any)
	if !ok {
		return fmt.Errorf("row value must be a map[string]any")
	}

	// Записываем данные
	tableData[key] = value
	print(operation + "\n")
	return nil
}

func (a *BaseStorage) rowIsExist(w http.ResponseWriter, r *http.Request, get bool) (bool, any) {
	var data RequestData
	if !processBodyToData(w, r, &data) {
		fmt.Println("Ошибка: не удалось распарсить JSON")
		return false, nil
	}

	if len(data.Data) != 1 {
		return false, nil
	}

	if data.NameBD == nil || data.NameTable == nil {
		fmt.Println("Ошибка: не указано имя БД или таблицы")
		return false, nil
	}

	// 1. Проверяем существование БД
	db, dbExists := a.memoryTables.storage[*data.NameBD]
	if !dbExists {
		fmt.Printf("БД '%s' не найдена\n", *data.NameBD)
		return false, nil
	}

	// 2. Проверяем существование таблицы
	table, tableExists := db[*data.NameTable]
	if !tableExists {
		fmt.Printf("Таблица '%s' не найдена в БД '%s'\n", *data.NameTable, *data.NameBD)
		return false, nil
	}

	// 3. Приводим к map[string]any (если ожидается именно такой тип)
	tableData, ok := table.(map[string]map[string]any)
	if !ok {
		fmt.Printf("Таблица '%s' не является map[string]any\n", *data.NameTable)
		return false, nil
	}

	var key string
	for k := range data.Data {
		key = k
		break
	}

	// 4. Проверяем существует ли ключ
	if tableData[key] != nil {
		if get {
			return true, tableData[key]
		} else {
			return true, nil
		}
	}

	return false, nil
}
