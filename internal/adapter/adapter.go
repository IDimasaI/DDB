// adapter/adapter.go
package adapter

import (
	"My-Redis/config"
	"My-Redis/storage"
	"net/http"
)

// Action представляет возможные действия адаптера
type Action int

const (
	GET Action = iota // начинаем с 0 и инкрементируем
	DELETE
	SET
	IsExist
	// можно добавить другие действия
)

type Adapter struct {
	Config  config.Config
	Storage storage.StorageActions
}

// New создает новый экземпляр адаптера
func Setup() *Adapter {
	config := config.GetMainConfig()
	var storageBD storage.StorageActions
	switch config.StorageType {
	case "base":
		storageBD = storage.NewBaseStorage(config)
	case "sqlite":
		storageBD = storage.NewSQLiteStorage(config)
	default:
		return nil
	}

	return &Adapter{
		Config:  config,
		Storage: storageBD,
	}
}

// InitContext инициализирует контекст
func (a *Adapter) InitContext() *storage.AppContext {
	return &storage.AppContext{}
}

// Handle обрабатывает HTTP запрос в зависимости от действия
func (a *Adapter) Handle(w http.ResponseWriter, r *http.Request, action Action, ctx *storage.AppContext) {
	switch action {
	case GET:
		a.Storage.GET(w, r, ctx)
		return
	case DELETE:
		a.Storage.DELETE(w, r, ctx)
	case IsExist:
		a.Storage.IsExist(w, r, ctx)
		return
	default:
		a.Storage.SET(w, r, ctx)
		return
	}
}
