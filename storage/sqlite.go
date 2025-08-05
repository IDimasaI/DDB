package storage

import (
	"My-Redis/config"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type sQLiteStorage struct {
	ConnectionDB *sql.DB
	NameDB       string
	config       config.Config
}

func NewSQLiteStorage(config config.Config) *sQLiteStorage {
	return &sQLiteStorage{
		ConnectionDB: nil,
		NameDB:       "",
		config:       config,
	}
}

func (a *sQLiteStorage) GET(w http.ResponseWriter, r *http.Request, ctx *AppContext) {
	event := ctx.Events
	event.EmitSync(nil, "event:onRequestStart")

	var body RequestData
	processBodyToData(w, r, &body)
	db := a.DB(*body.NameBD)
	query := fmt.Sprintf("SELECT * FROM %s", *body.NameTable)
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(name + "\n"))
	}
	event.EmitSync(nil, "event:onRequestEnd")
	defer r.Body.Close()
}

func (a *sQLiteStorage) SET(w http.ResponseWriter, r *http.Request, ctx *AppContext) {
	start := time.Now()

	var body RequestData
	processBodyToData(w, r, &body)
	dbOpenTime := time.Now()

	db := a.DB(*body.NameBD)

	tx, _ := db.Begin()
	txStart := time.Now()

	createNewTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT, name VARCHAR(255))", *body.NameTable)
	tx.Exec(createNewTable)

	prepare := fmt.Sprintf("INSERT INTO %s (name) VALUES (?)", *body.NameTable)
	stmt, _ := tx.Prepare(prepare)

	prepareTime := time.Now()

	stmt.Exec("dimasa")
	stmt.Close()
	execTime := time.Now()

	tx.Commit()
	commitTime := time.Now()

	fmt.Printf("\033[33m[SQLite:SET]\033[0m Timing: DB open=%v, Begin=%v, Prepare=%v, Exec=%v, Commit=%v, Total=%v\n",
		dbOpenTime.Sub(start),
		txStart.Sub(dbOpenTime),
		prepareTime.Sub(txStart),
		execTime.Sub(prepareTime),
		commitTime.Sub(execTime),
		commitTime.Sub(start),
	)
}

func (a *sQLiteStorage) DELETE(w http.ResponseWriter, r *http.Request, ctx *AppContext) {
	defer r.Body.Close()
}

func (a *sQLiteStorage) IsExist(w http.ResponseWriter, r *http.Request, ctx *AppContext) {
	defer r.Body.Close()
}

func (a *sQLiteStorage) DB(name string) *sql.DB {
	if a.NameDB == name {
		fmt.Println("Old Connection")
		return a.ConnectionDB
	} else {
		fmt.Println("New Connection")
		path := filepath.Join(a.config.PathEXE, a.config.PathStorage, name+".sqlite")

		var err error

		if _, err = os.Stat(a.config.PathStorage); os.IsNotExist(err) {
			err = os.MkdirAll(a.config.PathStorage, os.ModePerm)
			if err != nil {
				panic(err)
			}
		}

		if _, err = os.Stat(path); os.IsNotExist(err) {
			f, err := os.Create(path)
			if err != nil {
				panic(err)
			}
			f.Close()
		}

		if a.NameDB != "" {
			a.CloseDB()
		}

		a.ConnectionDB, err = sql.Open("sqlite", path)
		a.ConnectionDB.SetMaxOpenConns(1)
		if info, _ := os.Stat(path); info.Size() == 0 {
			a.ConnectionDB.Exec("PRAGMA journal_mode=WAL;")
		}

		if err != nil {
			panic(err)
		}
		a.NameDB = name
		return a.ConnectionDB
	}
}

func (a *sQLiteStorage) CloseDB() {
	a.ConnectionDB.Close()
}

func getKeyV2(data *map[string]any) (key string, value any, err error) {
	if len(*data) != 1 {
		return "", nil, fmt.Errorf("data содержет не [ключ]:Значение")
	} else {
		var key string
		var value any
		for k, v := range *data {
			key = k
			value = v
			break
		}
		return key, value, nil
	}
}
