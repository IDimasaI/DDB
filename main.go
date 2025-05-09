package main

import (
	"My-Redis/config"
	"My-Redis/internal/adapter"
	"My-Redis/internal/router"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {

	Config := config.GetMainConfig()

	path := ""
	if Config.IsDev {
		fmt.Println("Режим разработки...")

		port := flag.Int("port", 8188, "Port to scan")
		flag.Parse()

		path, _ = os.Getwd()
		path = filepath.Join(path)

		Config.Port = *port
		Config.PathEXE = path

		config.UpdateMainConfigInstance(Config)
	} else {
		fmt.Println("Режим продакшн...")
		path, _ = os.Executable()
		path = filepath.Join(filepath.Dir(path))

		Config.PathEXE = path

		config.UpdateMainConfigInstance(Config)
	}

	mux := router.NewMyRouter()
	mux.SetupUI(path)

	httpRouterConfig(&path, mux)
	httpRouterAdapter(path, mux)
	//if err := utils.Openbrowser("http://localhost:" + strconv.Itoa(Config.Port)); err != nil {
	//	log.Printf("Не удалось открыть браузер: %v", err)
	//}

	fmt.Println("Listening on http://localhost:" + strconv.Itoa(Config.Port))
	server := &http.Server{
		Addr:           ":" + strconv.Itoa(Config.Port),
		Handler:        Middleware(mux, router.MiddlewareOpt{MaxBytes: Config.MaxTransBytes})(mux),
		MaxHeaderBytes: 1 << 20, //128кб

	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}

func httpRouterConfig(path *string, mux *router.MyRouter) {
	mux.AddRouter("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			config.RefreshAdapterConfig()
		} else if r.Method == "GET" {

			filePath := filepath.Join(*path, "ui", "pages", "config.html")
			html, _ := os.ReadFile(filePath)

			config := config.GetMainConfig()

			pageData := struct {
				IsDev         bool
				MaxTransBytes int64
				StorageType   string
			}{
				IsDev:         config.IsDev,
				MaxTransBytes: config.MaxTransBytes,
				StorageType:   config.StorageType,
			}

			jsonData, err := json.Marshal(pageData)
			if err != nil {
				return
			}

			// Заменяем в HTML
			html = []byte(strings.Replace(
				string(html),
				`data-page="{}"`,
				`data-page='`+string(jsonData)+`'`,
				-1, // -1 означает замену всех вхождений
			))

			w.Write(html)

		}
	})
}

func Middleware(next http.Handler, options router.MiddlewareOpt) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверяем только POST/PUT/PATCH
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				r.Body = http.MaxBytesReader(w, r.Body, options.MaxBytes)
				r.ContentLength = 0
				// Если Content-Length превышает лимит — сразу отказываем
				if r.ContentLength > options.MaxBytes {
					http.Error(w, "Payload too large (max 1MB)", http.StatusRequestEntityTooLarge)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func httpRouterAdapter(path string, mux *router.MyRouter) {
	ad := adapter.New()
	mux.AddRouter("/set", func(w http.ResponseWriter, r *http.Request) {
		ad.Handle(w, r, adapter.SET)
		return
	})

	mux.AddRouter("/get", func(w http.ResponseWriter, r *http.Request) {
		ad.Handle(w, r, adapter.GET)
		return
	})
}
