package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prchop/chirpysrv/internal/auth"
	"github.com/prchop/chirpysrv/internal/database"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func appHandler(path string) http.Handler {
	return http.StripPrefix("/app", http.FileServer(http.Dir(path)))
}

type UserRequest struct {
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func newUserResponse(user database.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
}

func userHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var params UserRequest
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&params); err != nil {
			log.Printf("error decoding: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		if len(params.Password) == 0 || len(params.Email) == 0 {
			responseWithError(w, http.StatusBadRequest, "Empty field")
			return
		}

		password, err := auth.HashPassword(params.Password)
		if err != nil {
			log.Printf("error hashing password: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := cfg.db.CreateUser(r.Context(),
			database.CreateUserParams{
				Email:          params.Email,
				HashedPassword: password,
			},
		)
		if err != nil {
			log.Printf("error creating user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		createdUser := newUserResponse(dbUser)
		responseWithJSON(w, http.StatusCreated, createdUser)
	})
}

func userLoginHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var params UserRequest
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&params); err != nil {
			log.Printf("error decoding: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
		if err != nil {
			log.Printf("error getting user: %v", err)
			responseWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		if err := auth.CheckPasswordHash(params.Password, dbUser.HashedPassword); err != nil {
			log.Printf("error checking password: %v", err)
			responseWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		loggedInUser := newUserResponse(dbUser)
		responseWithJSON(w, http.StatusOK, loggedInUser)
	})
}

func updateUserHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		dbUser, err := cfg.db.UpdateUser(r.Context(),
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

		updatedUser := newUserResponse(dbUser)
		responseWithJSON(w, http.StatusOK, updatedUser)
	})
}

func getUsersHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbUsers, err := cfg.db.GetUsers(r.Context())
		if err != nil {
			log.Printf("error getting chirps: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		users := make([]UserResponse, len(dbUsers))

		for i, u := range dbUsers {
			users[i] = newUserResponse(u)
		}

		responseWithJSON(w, http.StatusOK, struct {
			Users []UserResponse `json:"users"`
		}{
			Users: users,
		})
	})
}

func getUserByIDHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing chirp id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := cfg.db.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("error getting chirp: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		fetchedUser := newUserResponse(dbUser)
		responseWithJSON(w, http.StatusOK, fetchedUser)
	})
}

func deleteUserByID(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing user id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := cfg.db.DeleteUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("error deleting user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, struct {
			DeletedUser UserResponse `json:"deleted_user"`
		}{
			DeletedUser: newUserResponse(dbUser),
		})
	})
}

type ChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func newChirpResponse(chirp database.Chirp) ChirpResponse {
	return ChirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
	}
}

func chirpHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		createdChirp := newChirpResponse(dbChirp)
		responseWithJSON(w, http.StatusCreated, createdChirp)
	})
}

func updateChirpHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		dbChirp, err := cfg.db.UpdateChirp(
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

		updatedChirp := newChirpResponse(dbChirp)
		responseWithJSON(w, http.StatusOK, updatedChirp)
	})
}

func getChirpsHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbChirps, err := cfg.db.GetChirps(r.Context())
		if err != nil {
			log.Printf("error getting chirps: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		chirps := make([]ChirpResponse, len(dbChirps))
		for i, c := range dbChirps {
			chirps[i] = newChirpResponse(c)
		}

		responseWithJSON(w, http.StatusOK, struct {
			Chrips []ChirpResponse `json:"chirps"`
		}{
			Chrips: chirps,
		})
	})
}

func getChripByIDHandler(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing chirp id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbChirp, err := cfg.db.GetChirpByID(r.Context(), chirpID)
		if err != nil {
			log.Printf("error getting chirp: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		fetchedChrip := newChirpResponse(dbChirp)
		responseWithJSON(w, http.StatusOK, fetchedChrip)
	})
}

func deleteChirpByID(cfg *apiConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing user id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbChirp, err := cfg.db.DeleteChirpByID(r.Context(), chirpID)
		if err != nil {
			log.Printf("error deleting user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, struct {
			DeletedChrip ChirpResponse `json:"deleted_chrip"`
		}{
			DeletedChrip: newChirpResponse(dbChirp),
		})
	})
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
	}{
		Error: msg},
	)
}
