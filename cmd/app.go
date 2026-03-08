package main

import (
	"net/http"

	"github.com/prathamesh/rate-limiter/internals/store"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

type Application struct {
	config *config
	store  *store.Storage
	logger *zap.SugaredLogger
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
	addr     string
	password string
	db       int
}

func (app *Application) getMuxHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		app.rateLimitChecker(w, r)
	})
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	return mux
}

func (app *Application) run() error {
	app.logger.Info("Starting server on ", app.config.addr)
	server := &http.Server{
		Addr:    app.config.addr,
		Handler: app.getMuxHandler(),
	}
	err := server.ListenAndServe()
	if err != nil {
		app.logger.Fatal("server failed to start: ", err)
	}
	return err
}
