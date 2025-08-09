package main

import (
	"log"
	"net/http"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func appHandler(path string) http.Handler {
	return http.StripPrefix("/app", http.FileServer(http.Dir(path)))
}

func main() {
	port := "8080"
	rootFilepath := "."
	cfg := NewAPIConfig()
	mux := http.NewServeMux()

	mw := func(h http.Handler) http.Handler {
		return cfg.MiddlewareMetricsInc(h)
	}

	// User endpoint
	mux.Handle("/app/", mw(appHandler(rootFilepath)))

	// API endpoint
	mux.Handle("GET /api/health", mw(http.HandlerFunc(healthHandler)))

	// Admin endpoint
	mux.Handle("GET /admin/metrics", cfg.HandlerMetrics())
	mux.Handle("POST /admin/reset", cfg.HandlerReset())

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	log.Printf("Chripy server start at http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
