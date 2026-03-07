package main

import (
	"net/http"

	"github.com/prathamesh/rate-limiter/internals/store"
)

type Application struct {
	config *config
	store  *store.Storage
}

type config struct {
	addr        string
	dbConfig    *dbConfig
	redisConfig *redisConfig
}	

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type redisConfig struct {
	addr string
	password string
	db   int
}

func (app *Application) getMuxHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		app.rateLimitChecker(w, r)
	})
	return mux
}

func (app *Application) run() error {
	server := &http.Server{
		Addr:    app.config.addr,
		Handler: app.getMuxHandler(),
	}
	return server.ListenAndServe()
}
