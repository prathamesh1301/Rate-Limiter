package main

import (
	"log"

	"github.com/prathamesh/rate-limiter/internals/db"
	"github.com/prathamesh/rate-limiter/internals/store"
)

func main() {
	dbConfig := &dbConfig{
		addr:         "postgres://admin:adminPassword@localhost:5432/postgres?sslmode=disable",
		maxOpenConns: 10,
		maxIdleConns: 5,
		maxIdleTime:  "5m",
	}
	db, err := db.New(dbConfig.addr, dbConfig.maxOpenConns, dbConfig.maxIdleConns, dbConfig.maxIdleTime)
	if err != nil {
		log.Fatal("failed to connect to database: ", err)
	}
	defer db.Close()
	storage := store.NewStorage(db)
	config := config{
		addr:     ":8080",
		dbConfig: dbConfig,
	}

	app := &Application{
		config: &config,
		store: storage,
	}
	app.run()
}
