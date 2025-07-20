package config

import (
	"My-Redis/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	IsDev         bool
	Port          int
	MaxTransBytes int64
	PathEXE       string
	StorageType   string
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig возвращает синглтон конфигурации
func GetMainConfig() Config {
	once.Do(func() {
		instance = &Config{}
		instance.IsDev = isDev()
		fmt.Println("isDev", instance.IsDev)
		var configPath string
		if instance.IsDev {
			wd, _ := os.Getwd()
			configPath = filepath.Join(wd, "config.json")
		} else {
			execPath, _ := os.Executable()
			configPath = filepath.Join(filepath.Dir(execPath), "config.json")
		}

		if err := instance.load(configPath); err != nil {
			fmt.Printf("Failed to read config file: %v\n", err)
		}
	})
	return *instance
}

func isDev() bool {
	temp, _ := os.Executable()
	fmt.Println(temp)
	return strings.Contains(temp, os.TempDir()) || strings.Contains(temp, "\\Local\\go-build")
}

func (c *Config) load(path string) error {
	json, err := utils.ReadJson[Config](path)
	if err != nil {
		return err
	}

	c.setPort(json.Port)
	c.IsDev = json.IsDev
	c.MaxTransBytes = json.MaxTransBytes
	c.PathEXE = json.PathEXE
	c.setStorageType(json.StorageType)
	return nil
}

func UpdateMainConfigInstance(cfg Config) {
	*instance = cfg
}

func UpdateMainConfigAll(cfg Config) {
	err := utils.WriteJson("config.json", cfg)
	if err != nil {
		fmt.Printf("Failed to write config file: %v\n", err)
	}
	// Обновляем синглтон
	*instance = cfg
}

func Log(cfg Config) {
	fmt.Printf("Config: %+v\n", cfg)
}

func (c *Config) setPort(port int) error {
	if port == 0 {
		print("Порт не может быть нулевым, ставим 8181")
		c.Port = 8181 // дефолтное значение
		return nil
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("порт %d невалиден (должен быть 1-65535)", port)
	}
	c.Port = port
	return nil
}

func (c *Config) setStorageType(storageType string) error {
	if storageType == "" {
		c.StorageType = "base"
		return nil
	} else {
		c.StorageType = storageType
	}
	return nil
}
