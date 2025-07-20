package storage

import (
	"My-Redis/core"
	"net/http"
)

type BdContext struct {
	Events *core.EventBus
}

type StorageActions interface {
	GET(w http.ResponseWriter, r *http.Request, ctx *BdContext)
	SET(w http.ResponseWriter, r *http.Request, ctx *BdContext)
	DELETE(w http.ResponseWriter, r *http.Request, ctx *BdContext)
	IsExist(w http.ResponseWriter, r *http.Request, ctx *BdContext)
}
