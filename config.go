package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/prchop/chirpysrv/internal/database"
)

type apiConfig struct {
	srvHits atomic.Int32
	db      *database.Queries
	secret  string
}

func (cfg *apiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		cfg.srvHits.Add(1)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(h)
}

func (cfg *apiConfig) HandlerMetrics() http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		result := cfg.srvHits.Load()
		html := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", result)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// w.Write([]byte(html))
		fmt.Fprint(w, html)
	}
	return http.HandlerFunc(h)
}

func (cfg *apiConfig) HandlerReset() http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {

		platform := os.Getenv("PLATFORM")
		if platform != "dev" {
			responseWithError(w, http.StatusForbidden, "Access denied")
			return
		}

		// reset metrics
		cfg.srvHits.Store(0)

		// delete all users
		if err := cfg.db.DeleteAllUsers(r.Context()); err != nil {
			log.Printf("error deleting users: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hits reset to 0\n"))
		w.Write([]byte("Users deleted\n"))
	}
	return http.HandlerFunc(h)
}

func NewAPIConfig(db *database.Queries) *apiConfig {
	return &apiConfig{srvHits: atomic.Int32{}, db: db}
}
