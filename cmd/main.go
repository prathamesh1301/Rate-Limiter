package main

import (
	"context"
	"log"

	"github.com/prathamesh/rate-limiter/internals/cache"
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

	redisConfig := &redisConfig{
		addr: "localhost:6379",
		password: "",
		db: 0,
	}

	db, err := db.New(dbConfig.addr, dbConfig.maxOpenConns, dbConfig.maxIdleConns, dbConfig.maxIdleTime)
	if err != nil {
		log.Fatal("failed to connect to database: ", err)
	}
	defer db.Close()
	ctx := context.Background()
	rdb, err := cache.NewRedisClient(ctx, redisConfig.addr, redisConfig.password, redisConfig.db)
	if err != nil {
		log.Fatal("failed to connect to redis: ", err)
	}
	defer rdb.Close()
	storage := store.NewStorage(db,rdb)
	config := config{	
		addr:        ":8080",
		dbConfig:    dbConfig,
		redisConfig: redisConfig,
	}

	app := &Application{
		config: &config,
		store:  storage,
	}
	app.run()
}
