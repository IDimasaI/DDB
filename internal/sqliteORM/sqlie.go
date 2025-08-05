package sqliteorm

import (
	"database/sql"
	"time"
	//_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	DB *sql.DB
}

type Timing struct {
	Start time.Time
	End   time.Time
}

func (db *DB) Close() {
	db.DB.Close()
}

func PrepareExec(db *DB, query string) (Timing, error) {
	start := time.Now()
	stmt, err := db.DB.Prepare(query)
	if err != nil {
		return Timing{}, err
	}
	defer stmt.Close()
	_, err = stmt.Exec()
	if err != nil {
		return Timing{}, err
	}
	end := time.Now()

	return Timing{
		Start: start,
		End:   end,
	}, nil
}
