package router

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

type MyRouter struct {
	mux *http.ServeMux
}
type MiddlewareOpt struct {
	MaxBytes int64
}

func NewMyRouter() *MyRouter {
	return &MyRouter{
		mux: http.NewServeMux(),
	}
}

// Статические файлы. и / страница
func (r *MyRouter) SetupUI(path string) {
	r.mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(
			http.Dir(filepath.Join(path, "ui", "static")),
		),
	))

	r.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("Hello World"))
	})
}

func (r *MyRouter) AddRouter(
	pattern string,
	handler func(w http.ResponseWriter, req *http.Request),
) {
	r.mux.HandleFunc(pattern, handler)
}

// Для совместимости с http.Handler
func (r *MyRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func CheckBodySize(w http.ResponseWriter, r *http.Request, maxBytes int64) bool {
	// 1. Проверяем Content-Length (если он указан)
	if r.ContentLength > maxBytes {
		http.Error(w, "Request too large", http.StatusBadRequest)
		return false
	}

	// 2. Если Content-Length не указан (или 0), проверяем тело вручную
	if r.ContentLength <= 0 {
		// Сохраняем оригинальное тело, чтобы потом восстановить
		originalBody := r.Body
		defer func() {
			if originalBody != nil {
				originalBody.Close()
			}
		}()

		// Создаем ограниченный reader
		limitedReader := http.MaxBytesReader(w, r.Body, maxBytes+1)

		// Читаем тело по кусочкам, но не сохраняем его
		buf := make([]byte, 32*1024) // 32KB буфер
		var totalRead int64

		for {
			n, err := limitedReader.Read(buf)
			totalRead += int64(n)

			if totalRead > maxBytes {
				http.Error(w, "Request too large", http.StatusBadRequest)
				return false
			}

			if err != nil {
				if err == io.EOF {
					break // Достигнут конец тела, размер в порядке
				}
				if strings.Contains(err.Error(), "request body too large") {
					http.Error(w, "Request too large", http.StatusBadRequest)
					return false
				}
				http.Error(w, "Error reading request body", http.StatusBadRequest)
				return false
			}
		}

		// Если размер в порядке, восстанавливаем тело
		r.Body = originalBody
		originalBody = nil // Чтобы defer не закрыл его
		return true

	}

	return true
}
