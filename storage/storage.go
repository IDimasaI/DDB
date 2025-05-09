package storage

import "net/http"

type StorageActions interface {
	GET(w http.ResponseWriter, r *http.Request)
	SET(w http.ResponseWriter, r *http.Request)
	DELETE(w http.ResponseWriter, r *http.Request)
}
