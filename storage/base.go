package storage

import (
	"My-Redis/config"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	storage     map[string]map[string]any
	currentSize int16
	maxTables   int16
	mu          sync.RWMutex
}

func NewMemoryTablesType(maxTables int16) memoryTablesType {
	return memoryTablesType{
		storage:     make(map[string]map[string]any, 1),
		currentSize: 0,
		maxTables:   maxTables,
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

func (a *BaseStorage) DELETE(w http.ResponseWriter, r *http.Request) {
	// 1. Проверяем существование данных
	var data RequestData
	processBodyToData(w, r, &data)
	if !a.memoryRowIsExist(false, &data) {
		http.Error(w, "Data not found", http.StatusNotFound)
		return
	}

	// 2. Получаем ключ для удаления
	key, err := getKey(data.Data)
	if err != nil {
		http.Error(w, "Invalid data key", http.StatusBadRequest)
		return
	}

	// 3. Проверяем существование таблицы и удаляем запись
	if data.NameBD != nil && data.NameTable != nil {
		table := a.memoryTables.storage[*data.NameBD][*data.NameTable].(map[string]map[string]any)
		if table != nil {
			delete(table, key)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Record with key '%s' deleted successfully", key)
			return
		}
	}

	http.Error(w, "Table not found", http.StatusNotFound)
}
func (a *BaseStorage) GET(w http.ResponseWriter, r *http.Request) {
	var data RequestData
	processBodyToData(w, r, &data)

	if !a.memoryRowIsExist(true, &data) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else {
		fmt.Println(data)
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

	err := a.addInMemory(*data.NameBD, *data.NameTable, data.Data)

	if a.memoryTables.currentSize > a.memoryTables.maxTables {
		a.addInFiles(*data.NameBD, *data.NameTable, data.Data, true)
		return
	}

	if err != nil {
		fmt.Println(err)
		return
	}

}

func (a *BaseStorage) IsExist(w http.ResponseWriter, r *http.Request) {
	var data RequestData
	processBodyToData(w, r, &data)

	if !a.memoryRowIsExist(false, &data) {
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
	key, err := getKey(row)
	if err != nil {
		return err
	}

	// Определяем операцию (добавление или изменение)
	operation := "added"
	if _, exists := tableData[key]; exists {
		operation = "updated"
	} else {
		a.memoryTables.currentSize = a.memoryTables.currentSize + 1
	}

	// Проверяем и приводим тип значения
	value, ok := row[key].(map[string]any)
	if !ok {
		return fmt.Errorf("row value must be a map[string]any")
	}

	// Записываем данные
	tableData[key] = value
	print(operation, "\n")
	return nil
}

func (a *BaseStorage) addInFiles(dbName, tableName string, row map[string]any, allMemory bool) error {
	basePath := filepath.Join(a.config.PathEXE, ".redis", dbName)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tablePath := filepath.Join(basePath, tableName+".db")
	file, err := os.OpenFile(tablePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return nil
}

// вызов фукнции для рекусивного использования addInFiles
func (a *BaseStorage) fullAddInMemoryToFiles() {

}

func createMetaInfoFile() {

}

func (a *BaseStorage) memoryRowIsExist(get bool, data *RequestData) bool {

	if data.NameBD == nil || data.NameTable == nil {
		fmt.Println("Ошибка: не указано имя БД или таблицы")
		return false
	}

	// 1. Проверяем существование БД
	db, dbExists := a.memoryTables.storage[*data.NameBD]
	if !dbExists {
		fmt.Printf("БД '%s' не найдена\n", *data.NameBD)
		return false
	}

	// 2. Проверяем существование таблицы
	table, tableExists := db[*data.NameTable]
	if !tableExists {
		fmt.Printf("Таблица '%s' не найдена в БД '%s'\n", *data.NameTable, *data.NameBD)
		return false
	}

	// 3. Приводим к map[string]any (если ожидается именно такой тип)
	tableData, ok := table.(map[string]map[string]any)
	if !ok {
		fmt.Printf("Таблица '%s' не является map[string]any\n", *data.NameTable)
		return false
	}

	key, err := getKey(data.Data)
	if err != nil {
		return false
	}

	// 4. Проверяем существует ли ключ
	if val, exists := tableData[key]; exists {
		if get {
			data.Data = val
		}
		return true
	}

	return false
}

func getKey(data map[string]any) (key string, err error) {
	if len(data) != 1 {
		return "", fmt.Errorf("data содержет не [ключ]:Значение")
	} else {
		var key string
		for k := range data {
			key = k
			break
		}
		return key, nil
	}
}
