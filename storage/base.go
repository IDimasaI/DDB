package storage

import (
	"My-Redis/config"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
type baseStorage struct {
	config       config.Config
	memoryTables memoryTablesType
}

// Инициализация выбранного БД(только 1 раз)
func NewBaseStorage(cfg config.Config) *baseStorage {
	return &baseStorage{
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

const (
	SizeMetaPage = 4 * 1024 // 4 КБ на метаданные
	SizePages    = 8 * 1024 // 8 КБ на таблицу
)

func (a *baseStorage) DELETE(w http.ResponseWriter, r *http.Request, ctx *AppContext) {
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
func (a *baseStorage) GET(w http.ResponseWriter, r *http.Request, ctx *AppContext) {
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

func (a *baseStorage) SET(w http.ResponseWriter, r *http.Request, ctx *AppContext) {
	var data RequestData
	_ = processBodyToData(w, r, &data)

	if data.NameBD == nil || data.NameTable == nil {
		http.Error(w, `{"status":"error","message":"Missing required fields"}`, http.StatusBadRequest)
		return
	}

	err := a.addInMemory(*data.NameBD, *data.NameTable, data.Data)

	if a.memoryTables.currentSize > a.memoryTables.maxTables {
		err = a.addInFiles(*data.NameBD, *data.NameTable, data.Data, true)
		if err != nil {
			fmt.Println(err)
			return
		}

		return
	}

	if err != nil {
		fmt.Println(err)
		return
	}

}

// Проверка существования строки в памяти
// Если найден - возвращаем true и 200
// Если нет - возвращаем false и 404
func (a *baseStorage) IsExist(w http.ResponseWriter, r *http.Request, ctx *AppContext) {
	var data RequestData
	processBodyToData(w, r, &data)

	if !a.memoryRowIsExist(false, &data) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func processBodyToData(w http.ResponseWriter, r *http.Request, data *RequestData) bool {
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		http.Error(w, `{"status":"error","message":"Invalid JSON"}`, http.StatusBadRequest)
		return false
	}
	defer r.Body.Close()
	return true
}

type metaData struct {
	Keys    map[string]int64 `json:"keys"`    // Список ключей и строк
	Pages   int32            `json:"pages"`   // Количество строк
	Version int              `json:"version"` // Версия
	Voids   []int32          `json:"voids"`
}

func (a *baseStorage) addInMemory(dbName, tableName string, row map[string]any) error {
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

func (a *baseStorage) addInFiles(dbName, tableName string, row map[string]any, allMemory bool) error {
	basePath := filepath.Join(a.config.PathEXE, ".redis", dbName)

	if allMemory {
		//TDD
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tablePath := filepath.Join(basePath, tableName+".db")

	// Проверяем, существует ли файл и создаем его с метаданными, если нет
	if _, err := os.Stat(tablePath); os.IsNotExist(err) {
		if err := createNewTableFile(tablePath); err != nil {
			return err
		}
	}

	rowKey, err := getKey(row)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(tablePath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for reading: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	meta, err := scanMetaInfoFile(scanner, file)
	if err != nil {
		return err
	}

	if meta.Keys == nil || meta.Keys[rowKey] == 0 {
		rowNum, err := writeNewKey(scanner, row, file, &meta)
		if err != nil {
			return err
		}
		meta.Keys[rowKey] = rowNum
	}

	fmt.Println(meta)
	// Записываем метаданные
	updatedMetaData(file, meta)

	if err = scanner.Err(); err != nil {
		return fmt.Errorf("error while scanning file: %w", err)
	}

	return nil
}

// вызов фукнции для рекусивного использования addInFiles
func (a *baseStorage) fullAddInMemoryToFiles() {

}

// заполнитель на n байт
func fillBuffer(size int) []byte {
	return make([]byte, size)
}

func createNewTableFile(tablePath string) error {
	file, err := os.Create(tablePath)
	if err != nil {
		return fmt.Errorf("failed to create table file: %w", err)
	}
	defer file.Close()

	// Записываем метаданные и делаем заполнитель на 4кб
	meta, _ := json.Marshal(metaData{Pages: 0, Keys: map[string]int64{}, Version: 1, Voids: []int32{}})
	metalen := len(meta) + len("\n")
	if _, err := file.WriteString(
		string(meta) + string(fillBuffer(SizeMetaPage-metalen)) + "\n",
	); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func scanMetaInfoFile(scanner *bufio.Scanner, file *os.File) (metaData, error) {
	_, _ = file.Seek(0, io.SeekStart)
	var meta metaData
	if scanner.Scan() {
		line := scanner.Bytes()
		line = bytes.TrimRight(line, "\x00")
		json.Unmarshal(line, &meta)
		return meta, nil
	}
	return metaData{}, scanner.Err()
}

func updatedMetaData(file *os.File, metaData metaData) error {
	filepos, _ := file.Seek(0, io.SeekCurrent)

	_, _ = file.Seek(0, SizeMetaPage)

	metaNew, err := json.Marshal(metaData)
	if err != nil {
		return err
	}
	metaNew = append(metaNew, fillBuffer(SizeMetaPage-len(metaNew))...)
	_, err = file.WriteAt(metaNew, 0)
	if err != nil {
		return err
	}
	_, err = file.Seek(filepos, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

func writeNewKey(scanner *bufio.Scanner, row map[string]any, file *os.File, metaData *metaData) (rowNum int64, error error) {
	if metaData.Voids == nil {
	}
	rowNum, _ = file.Seek(0, io.SeekEnd)
	data := fmt.Sprintf("%v\n", row)
	if _, err := file.WriteString(data); err != nil {
		return 0, err
	}
	return rowNum, nil
}

func (a *baseStorage) memoryRowIsExist(get bool, data *RequestData) bool {

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
