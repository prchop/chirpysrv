package main

import (
	"log"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		log.Fatal("Error parsing config", err)
	}

	port := cfg.Port
	mux := http.NewServeMux()
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatal("Error starting app", err)
	}

	mw := func(h http.Handler) http.Handler {
		return app.MiddlewareMetricsInc(h)
	}

	mux.Handle("/app/", mw(appHandler(".")))

	mux.Handle("GET /api/health", mw(http.HandlerFunc(healthHandler)))

	mux.Handle("GET /api/users", mw(getUsersHandler(app)))
	mux.Handle("GET /api/users/{id}", mw(getUserByIDHandler(app)))

	mux.Handle("GET /api/chirps", mw(getChirpsHandler(app)))
	mux.Handle("GET /api/chirps/{id}", mw(getChripByIDHandler(app)))

	mux.Handle("POST /api/users", mw(userHandler(app)))
	mux.Handle("POST /api/login", mw(userLoginHandler(app)))
	mux.Handle("POST /api/chirps", mw(chirpHandler(app)))
	mux.Handle("POST /api/refresh", mw(refreshHandler(app)))
	mux.Handle("POST /api/revoke", mw(revokeHandler(app)))

	mux.Handle("PUT /api/users", mw(updateUserHandler(app)))
	mux.Handle("PATCH /api/chirps/{id}", mw(updateChirpHandler(app)))

	mux.Handle("DELETE /api/users/{id}", mw(deleteUserByID(app)))
	mux.Handle("DELETE /api/chirps/{chirpID}", mw(deleteChirpByID(app)))

	mux.Handle("GET /admin/metrics", app.HandlerMetrics())
	mux.Handle("POST /admin/reset", app.HandlerReset())

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	log.Printf("Chirpy server start at http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
