package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// readFile читает содержимое файла и возвращает его в виде []byte.
// Если происходит ошибка, возвращает её вторым аргументом.
func ReadFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// ReadJson читает JSON-файл и возвращает данные в указанном типе
func ReadJson[T any](path string) (T, error) {
	var result T

	fileData, err := ReadFile(path)
	if err != nil {
		return result, err
	}

	if !json.Valid(fileData) {
		return result, fmt.Errorf("ошибка, это не json")
	}

	err = json.Unmarshal(fileData, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func WriteJson[T any](path string, data T) error {
	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, file, 0644)
}
