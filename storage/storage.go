package storage

import (
	"My-Redis/core/Events"
	"net/http"
)

type AppContext struct {
	Events *Events.EventBus
}

type StorageActions interface {
	GET(w http.ResponseWriter, r *http.Request, ctx *AppContext)
	SET(w http.ResponseWriter, r *http.Request, ctx *AppContext)
	DELETE(w http.ResponseWriter, r *http.Request, ctx *AppContext)
	IsExist(w http.ResponseWriter, r *http.Request, ctx *AppContext)
}
