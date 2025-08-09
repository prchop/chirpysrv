package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fsrvHits atomic.Int32
}

func (cfg *apiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		cfg.fsrvHits.Add(1)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(h)
}

func (cfg *apiConfig) HandlerMetrics() http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		result := cfg.fsrvHits.Load()
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
		cfg.fsrvHits.Store(0)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hits reset to 0\n"))
	}
	return http.HandlerFunc(h)
}

func NewAPIConfig() *apiConfig {
	return &apiConfig{fsrvHits: atomic.Int32{}}
}
