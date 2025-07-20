package storage

import (
	"My-Redis/config"
	"fmt"
	"net/http"
)

type sQLiteStorage struct {
	config config.Config
}

func NewSQLiteStorage(config config.Config) *sQLiteStorage {
	return &sQLiteStorage{
		config: config,
	}
}

func (a *sQLiteStorage) GET(w http.ResponseWriter, r *http.Request, ctx *BdContext) {
	event := ctx.Events
	event.EmitSync(nil, "event:onRequestStart")
	fmt.Println(event.SyncEvents)

	event.EmitSync(nil, "event:onRequestEnd")
	defer r.Body.Close()
}

func (a *sQLiteStorage) SET(w http.ResponseWriter, r *http.Request, ctx *BdContext) {
	defer r.Body.Close()
}

func (a *sQLiteStorage) DELETE(w http.ResponseWriter, r *http.Request, ctx *BdContext) {
	defer r.Body.Close()
}

func (a *sQLiteStorage) IsExist(w http.ResponseWriter, r *http.Request, ctx *BdContext) {
	defer r.Body.Close()
}
