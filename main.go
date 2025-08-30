package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/prchop/chirpysrv/internal/database"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func userHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		var u User
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&u); err != nil {
			log.Printf("error decoding: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := cfg.db.CreateUser(r.Context(), u.Email)
		if err != nil {
			log.Printf("error creating user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
		}

		responseWithJSON(w, http.StatusCreated, dbUser)
	}
	return http.HandlerFunc(h)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func appHandler(path string) http.Handler {
	return http.StripPrefix("/app", http.FileServer(http.Dir(path)))
}

// TODO:
// Replace any profane words with 4 asterisk
// Profane words:
// - kerfuffle
// - sharbert
// - fornax
func validateHandler(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Body string `json:"body"`
	}

	var params requestBody
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&params); err != nil {
		log.Printf("error decoding: %v", err)
		responseWithError(w, http.StatusBadRequest, "Something went wrong")
		// responseWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(params.Body) == 0 {
		responseWithError(w, http.StatusBadRequest, "Empty request body")
		return
	}

	if len(params.Body) > 140 {
		responseWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	spl := strings.Split(params.Body, " ")
	for i := range spl {
		s := strings.ToLower(spl[i])
		if s == "kerfuffle" || s == "sharbert" || s == "fornax" {
			spl[i] = "****"
		}
	}
	str := strings.Join(spl, " ")

	responseWithJSON(w, http.StatusOK, struct {
		Body string `json:"cleaned_body"`
	}{Body: str})
}

func responseWithJSON(w http.ResponseWriter, code int, payload any) {
	resp, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshaling: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Something went wrong"}`))
		// w.Write([]byte(`{"error":"Internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(resp)
}

func responseWithError(w http.ResponseWriter, code int, msg string) {
	responseWithJSON(w, code, struct {
		Error string `json:"error"`
	}{Error: msg})
}

func main() {
	port := "8080"
	rootFilepath := "."
	mux := http.NewServeMux()

	dbURI := os.Getenv("GOOSE_DBSTRING")
	dbDriver := os.Getenv("GOOSE_DRIVER")
	db, err := sql.Open(dbDriver, dbURI)
	if err != nil {
		log.Print(err)
	}

	dbQueries := database.New(db)
	cfg := NewAPIConfig(dbQueries)
	// cfg = &apiConfig{fsrvHits: atomic.Int32{}, db: dbQueries}

	mw := func(h http.Handler) http.Handler {
		return cfg.MiddlewareMetricsInc(h)
	}

	// User endpoint
	mux.Handle("/app/", mw(appHandler(rootFilepath)))

	// API endpoint
	mux.Handle("GET /api/health", mw(http.HandlerFunc(healthHandler)))
	mux.Handle("POST /api/validate_chirp", mw(http.HandlerFunc(validateHandler)))
	mux.Handle("POST /api/users", mw(userHandler(cfg)))

	// Admin endpoint
	mux.Handle("GET /admin/metrics", cfg.HandlerMetrics())
	mux.Handle("POST /admin/reset", cfg.HandlerReset())

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	log.Printf("Chripy server start at http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
