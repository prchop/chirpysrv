package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/prchop/chirpysrv/internal/database"
)

type Config struct {
	Port      string `env:"PORT"`
	DBDriver  string `env:"DB_DRIVER"`
	DBURI     string `env:"DB_URI"`
	Platform  string `env:"PLATFORM"`
	JWTSecret string `env:"JWT_SECRET"`
	PolkaKey  string `env:"POLKA_KEY"`
}

type App struct {
	db      *database.Queries
	srvHits atomic.Int32
	config  Config
}

func (app *App) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.srvHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (app *App) HandlerMetrics() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := app.srvHits.Load()
		templ := `    <html>
    <body><h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p></body>
    </html>`
		html := fmt.Sprintf(templ, result)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})
}

func (app *App) HandlerReset() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		platform := app.config.Platform
		if platform != "dev" {
			responseWithError(w, http.StatusForbidden, "Access denied")
			return
		}

		// reset metrics
		app.srvHits.Store(0)

		// delete all users
		if err := app.db.DeleteAllUsers(r.Context()); err != nil {
			log.Printf("error deleting users: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hits reset to 0\n"))
		w.Write([]byte("Users deleted\n"))
	})
}

func NewApp(cfg Config) (*App, error) {
	db, err := sql.Open(cfg.DBDriver, cfg.DBURI)
	if err != nil {
		return nil, err
	}
	queries := database.New(db)

	return &App{
		db:      queries,
		srvHits: atomic.Int32{},
		config:  cfg,
	}, nil
}
