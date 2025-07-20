package main

import (
	"My-Redis/config"
	"My-Redis/core"
	"My-Redis/internal/adapter"
	"My-Redis/internal/router"
	"My-Redis/internal/utils"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {

	Config := config.GetMainConfig()

	path := ""
	{

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

			path, _ = os.Executable()
			path = filepath.Join(filepath.Dir(path))

			Config.PathEXE = path

			config.UpdateMainConfigInstance(Config)
		}
	}

	mux := router.NewMyRouter()
	mux.SetupUI(path)

	//Инициализация роутеров UI
	httpRouterConfig(&path, mux)

	ad := adapter.Setup()
	ctx := ad.InitContext()
	ctx.Events = core.NewEventBus()

	//Инициализация событий
	{
		ctx.Events.AddSyncHandler("event:ServerStarted", core.EventHandler{
			Func: func(event *core.Event) {
				fmt.Println("Listening on http://localhost:" + strconv.Itoa(Config.Port))
			},
			Priority: 1,
		})

		ctx.Events.AddSyncHandler("event:onRequestStart", core.EventHandler{
			Func: func(event *core.Event) {
				fmt.Println("Request Get in:" + time.Now().String())
			},
			Priority: 1,
		})

		ctx.Events.AddSyncHandler("event:onRequestEnd", core.EventHandler{
			Func: func(event *core.Event) {
				fmt.Println("Request Get out:" + time.Now().String())
			},
			Priority: 1,
		})
	}

	//Инициализация калбеков БД
	{
		mux.AddRouter("/set", func(w http.ResponseWriter, r *http.Request) {
			ad.Handle(w, r, adapter.SET, ctx)
		})

		mux.AddRouter("/get", func(w http.ResponseWriter, r *http.Request) {
			ad.Handle(w, r, adapter.GET, ctx)
		})

		mux.AddRouter("/delete", func(w http.ResponseWriter, r *http.Request) {
			ad.Handle(w, r, adapter.DELETE, ctx)
		})

		mux.AddRouter("/IsExist", func(w http.ResponseWriter, r *http.Request) {
			ad.Handle(w, r, adapter.IsExist, ctx)
		})
	}

	//Открываем браузер на странице с конфигом
	if err := utils.Openbrowser("http://localhost:" + strconv.Itoa(Config.Port) + "/config"); err != nil {
		log.Printf("Не удалось открыть браузер: %v", err)
	}

	//Событие старта сервера

	server := &http.Server{
		Addr:           ":" + strconv.Itoa(Config.Port),
		Handler:        router.Middleware(mux, router.MiddlewareOpt{MaxBytes: Config.MaxTransBytes})(mux),
		MaxHeaderBytes: 1 << 20, //128кб
	}

	ctx.Events.EmitSync(nil, "event:ServerStarted")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}

func httpRouterConfig(path *string, mux *router.MyRouter) {
	mux.AddRouter("/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			config.RefreshAdapterConfig()
		case "GET":

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
			defer r.Body.Close()
		}
	})
}
