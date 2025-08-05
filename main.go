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
	mw := func(h http.Handler) http.Handler {
		return cfg.MiddlewareMetricsInc(h)
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", mw(appHandler(rootFilepath)))
	mux.Handle("GET /health", mw(http.HandlerFunc(healthHandler)))
	mux.Handle("GET /metrics", cfg.HandlerMetrics())
	mux.Handle("POST /reset", cfg.HandlerReset())

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	log.Printf("Chripy server start at http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
