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
	// можно добавить другие действия
)

type Adapter struct {
	Config  config.Config
	Storage storage.StorageActions
}

// New создает новый экземпляр адаптера
func New() *Adapter {
	config := config.GetMainConfig()
	var storageBD storage.StorageActions
	switch config.StorageType {
	case "base":
		storageBD = storage.NewBaseStorage(config)
	//case "memory": ....
	default:
		return nil
	}

	return &Adapter{
		Config:  config,
		Storage: storageBD,
	}
}

// Handle обрабатывает HTTP запрос в зависимости от действия
func (a *Adapter) Handle(w http.ResponseWriter, r *http.Request, action Action) {
	switch action {
	case GET:
		a.Storage.GET(w, r)
		return
	default:
		a.Storage.SET(w, r)
		return
	}
}
