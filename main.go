package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	_ "github.com/lib/pq"
	"github.com/prchop/chirpysrv/internal/database"
)

func main() {
	port := ":8080"
	rootFilepath := "."
	mux := http.NewServeMux()
	secret := os.Getenv("JWT_SECRET")

	dbURI := os.Getenv("GOOSE_DBSTRING")
	dbDriver := os.Getenv("GOOSE_DRIVER")
	db, err := sql.Open(dbDriver, dbURI)
	if err != nil {
		log.Print(err)
	}

	dbQueries := database.New(db)

	cfg := &apiConfig{
		srvHits: atomic.Int32{},
		db:      dbQueries,
		secret:  secret,
	}

	mw := func(h http.Handler) http.Handler {
		return cfg.MiddlewareMetricsInc(h)
	}

	mux.Handle("/app/", mw(appHandler(rootFilepath)))

	mux.Handle("GET /api/health", mw(http.HandlerFunc(healthHandler)))

	mux.Handle("GET /api/users", mw(getUsersHandler(cfg)))
	mux.Handle("GET /api/users/{id}", mw(getUserByIDHandler(cfg)))

	mux.Handle("GET /api/chirps", mw(getChirpsHandler(cfg)))
	mux.Handle("GET /api/chirps/{id}", mw(getChripByIDHandler(cfg)))

	mux.Handle("POST /api/users", mw(userHandler(cfg)))
	mux.Handle("POST /api/login", mw(userLoginHandler(cfg)))
	mux.Handle("POST /api/chirps", mw(chirpHandler(cfg)))

	mux.Handle("PATCH /api/users/{id}", mw(updateUserHandler(cfg)))
	mux.Handle("PATCH /api/chirps/{id}", mw(updateChirpHandler(cfg)))

	mux.Handle("DELETE /api/users/{id}", mw(deleteUserByID(cfg)))
	mux.Handle("DELETE /api/chirps/{id}", mw(deleteChirpByID(cfg)))

	mux.Handle("GET /admin/metrics", cfg.HandlerMetrics())
	mux.Handle("POST /admin/reset", cfg.HandlerReset())

	srv := &http.Server{Addr: port, Handler: mux}

	log.Printf("Chirpy server start at http://localhost%s", port)
	log.Fatal(srv.ListenAndServe())
}
