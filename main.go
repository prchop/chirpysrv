package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/prchop/chirpysrv/internal/database"
)

func userHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		type CreateUserRequest struct {
			Email string `json:"email"`
		}

		var params CreateUserRequest
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&params); err != nil {
			log.Printf("error decoding: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := cfg.db.CreateUser(r.Context(), params.Email)
		if err != nil {
			log.Printf("error creating user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
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

func chirpHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		type CreateChirpRequest struct {
			Body   string    `json:"body"`
			UserID uuid.UUID `json:"user_id"`
		}

		var params CreateChirpRequest
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&params); err != nil {
			log.Printf("error decoding: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
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

		dbChirp, err := cfg.db.CreateChirp(
			r.Context(),
			database.CreateChirpParams{
				Body:   str,
				UserID: params.UserID,
			},
		)
		if err != nil {
			log.Printf("error creating chirps: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusCreated, dbChirp)
	}
	return http.HandlerFunc(h)
}

func updateUserHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		type requestUpdateUser struct {
			Email string `json:"email"`
		}

		var params requestUpdateUser
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&params); err != nil {
			log.Printf("error decoding: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		parsedID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		updatedUser, err := cfg.db.UpdateUser(
			r.Context(),
			database.UpdateUserParams{
				Email: params.Email,
				ID:    parsedID,
			},
		)
		if err != nil {
			log.Printf("error updating user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, updatedUser)
	}

	return http.HandlerFunc(h)
}

func updateChirpHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		type requestUpdateChirp struct {
			Body string `json:"body"`
		}

		var params requestUpdateChirp
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&params); err != nil {
			log.Printf("error decoding: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
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

		parsedID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		updatedChirp, err := cfg.db.UpdateChirp(
			r.Context(),
			database.UpdateChirpParams{
				Body: str,
				ID:   parsedID,
			},
		)
		if err != nil {
			log.Printf("error updating chrip: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, updatedChirp)
	}

	return http.HandlerFunc(h)
}

func getUsersHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		users, err := cfg.db.GetUsers(r.Context())
		if err != nil {
			log.Printf("error getting chirps: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, users)
	}

	return http.HandlerFunc(h)
}

func getUserByIDHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing chirp id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		user, err := cfg.db.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("error getting chirp: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, user)
	}

	return http.HandlerFunc(h)
}

func deleteUserByID(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing user id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		deletedUser, err := cfg.db.DeleteUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("error deleting user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, struct {
			DeletedUser database.User `json:"deleted_user"`
		}{DeletedUser: deletedUser})
	}

	return http.HandlerFunc(h)
}

func getChirpsHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		chirps, err := cfg.db.GetChirps(r.Context())
		if err != nil {
			log.Printf("error getting chirps: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, chirps)
	}

	return http.HandlerFunc(h)
}

func getChripByIDHandler(cfg *apiConfig) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing chirp id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		chirp, err := cfg.db.GetChirpByID(r.Context(), chirpID)
		if err != nil {
			log.Printf("error getting chirp: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, chirp)
	}

	return http.HandlerFunc(h)
}

func responseWithJSON(w http.ResponseWriter, code int, payload any) {
	resp, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshaling: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal server error"}`))
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

	mw := func(h http.Handler) http.Handler {
		return cfg.MiddlewareMetricsInc(h)
	}

	// App endpoint
	mux.Handle("/app/", mw(appHandler(rootFilepath)))

	// API endpoint
	mux.Handle("GET /api/health", mw(http.HandlerFunc(healthHandler)))

	mux.Handle("GET /api/users", mw(getUsersHandler(cfg)))
	mux.Handle("GET /api/users/{id}", mw(getUserByIDHandler(cfg)))

	mux.Handle("GET /api/chirps", mw(getChirpsHandler(cfg)))
	mux.Handle("GET /api/chirps/{id}", mw(getChripByIDHandler(cfg)))

	mux.Handle("POST /api/chirps", mw(chirpHandler(cfg)))
	mux.Handle("POST /api/users", mw(userHandler(cfg)))

	mux.Handle("PATCH /api/users/{id}", mw(updateUserHandler(cfg)))
	mux.Handle("PATCH /api/chirps/{id}", mw(updateChirpHandler(cfg)))

	mux.Handle("DELETE /api/users/{id}", mw(deleteUserByID(cfg)))
	// Admin endpoint
	mux.Handle("GET /admin/metrics", cfg.HandlerMetrics())
	mux.Handle("POST /admin/reset", cfg.HandlerReset())

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	log.Printf("Chirpy server start at http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
