package main

import (
	"encoding/json"
	"log"
	"net/http"
	"reflect"
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

func refreshHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reftoken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("error retrieving refresh token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := app.db.GetUserByRefreshToken(r.Context(), reftoken)
		if err != nil {
			log.Printf("error retrieving user with refresh token: %v", err)
			responseWithError(w, http.StatusUnauthorized, "Token didn't exist or already expired")
			return
		}

		token, err := auth.MakeJWT(dbUser.ID, app.config.JWTSecret, time.Hour)
		if err != nil {
			log.Printf("error generating access token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		responseWithJSON(w, http.StatusOK, struct {
			Token string `json:"token"`
		}{
			Token: token,
		})
	})
}

func revokeHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reftoken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("error retrieving refresh token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		if _, err = app.db.RevokeRefreshToken(r.Context(), reftoken); err != nil {
			log.Printf("error revoking refresh token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Token didn't exist or already revoked")
			return
		}

		responseWithNoContent(w, http.StatusNoContent)
	})
}

type UserRequest struct {
	Password string `json:"password" required:"true"`
	Email    string `json:"email" required:"true"`
}

func validate(s any) map[string]string {
	errors := make(map[string]string)

	t := reflect.TypeOf(s)
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Tag.Get("required") == "true" {
			value := reflect.ValueOf(s).Field(i).String()
			if value == "" {
				key := strings.ToLower(field.Name)
				errors[key] = key + " is required"
			}
		}
	}

	return errors
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

func newUserResponse(user database.User) UserResponse {
	return UserResponse{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
}

type AuthResponse struct {
	UserResponse
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func newAuthResponse(user UserResponse, token, reftoken string) AuthResponse {
	return AuthResponse{
		UserResponse: user,
		Token:        token,
		RefreshToken: reftoken,
	}
}

func userHandler(app *App) http.Handler {
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

		errors := validate(params)
		if len(errors) > 0 {
			responseWithValidationError(w, http.StatusBadRequest, "user validation failed", errors)
			return
		}

		password, err := auth.HashPassword(params.Password)
		if err != nil {
			log.Printf("error hashing password: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := app.db.CreateUser(r.Context(),
			database.CreateUserParams{
				Email:          params.Email,
				HashedPassword: password,
			},
		)
		if err != nil {
			log.Printf("error generating user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		createdUser := newUserResponse(dbUser)
		responseWithJSON(w, http.StatusCreated, createdUser)
	})
}

func userLoginHandler(app *App) http.Handler {
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

		errors := validate(params)
		if len(errors) > 0 {
			responseWithValidationError(w, http.StatusBadRequest, "user validation failed", errors)
			return
		}

		dbUser, err := app.db.GetUserByEmail(r.Context(), params.Email)
		if err != nil {
			log.Printf("error retrieving user: %v", err)
			responseWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		if err = auth.CheckPasswordHash(params.Password, dbUser.HashedPassword); err != nil {
			log.Printf("error checking password: %v", err)
			responseWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		// token should expire after 1 hour
		token, err := auth.MakeJWT(dbUser.ID, app.config.JWTSecret, time.Hour)
		if err != nil {
			log.Printf("error generating access token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		reftoken, err := auth.MakeRefreshToken()
		if err != nil {
			log.Printf("error generating refresh token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbRefToken, err := app.db.CreateRefreshToken(
			r.Context(),
			database.CreateRefreshTokenParams{
				Token:     reftoken,
				UserID:    dbUser.ID,
				ExpiresAt: time.Now().AddDate(0, 0, 60),
			})
		if err != nil {
			log.Printf("error retrieving user: %v", err)
			responseWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		loggedInUser := newUserResponse(dbUser)
		userWithAuth := newAuthResponse(loggedInUser, token, dbRefToken.Token)

		responseWithJSON(w, http.StatusOK, userWithAuth)
	})
}

func updateUserHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type requestUpdateUser struct {
			Password string `json:"password" required:"true"`
			Email    string `json:"email" required:"true"`
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

		errors := validate(params)
		if len(errors) > 0 {
			responseWithValidationError(w, http.StatusBadRequest, "user validation failed", errors)
			return
		}

		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("error parsing: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		validID, err := auth.ValidateJWT(token, app.config.JWTSecret)
		if err != nil {
			log.Printf("error checking token: %v", err)
			responseWithError(w, http.StatusUnauthorized, "The provided token is invalid or missing")
			return
		}

		password, err := auth.HashPassword(params.Password)
		if err != nil {
			responseWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		dbUser, err := app.db.UpdateUser(r.Context(), database.UpdateUserParams{
			Email:          params.Email,
			HashedPassword: password,
			ID:             validID,
		})
		if err != nil {
			log.Printf("error updating user: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		if err = auth.CheckPasswordHash(params.Password, dbUser.HashedPassword); err != nil {
			log.Printf("error checking password: %v", err)
			responseWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		updatedUser := newUserResponse(dbUser)
		responseWithJSON(w, http.StatusOK, updatedUser)
	})
}

func getUsersHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbUsers, err := app.db.GetUsers(r.Context())
		if err != nil {
			log.Printf("error retrieving chirps: %v", err)
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

func getUserByIDHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing chirp id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := app.db.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("error retrieving chirp: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		fetchedUser := newUserResponse(dbUser)
		responseWithJSON(w, http.StatusOK, fetchedUser)
	})
}

func deleteUserByID(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing user id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbUser, err := app.db.DeleteUserByID(r.Context(), userID)
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

func upgradeUserHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type requestUpgradeUser struct {
			Event string `json:"event"`
			Data  struct {
				UserID uuid.UUID `json:"user_id"`
			} `json:"data"`
		}
		var params requestUpgradeUser
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&params); err != nil {
			log.Printf("error decoding params: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		key, err := auth.GetAPIKey(r.Header)
		if err != nil {
			log.Printf("error retrieving api key: %v", err)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if key != app.config.PolkaKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if params.Event != "user.upgrade" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		_, err = app.db.UpgradeUser(r.Context(), database.UpgradeUserParams{
			IsChirpyRed: true,
			ID:          params.Data.UserID,
		})
		if err != nil {
			log.Printf("error upgrading user: %v", err)
			responseWithError(w, http.StatusNotFound, "User not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
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
		UserID:    chirp.UserID,
	}
}

func chirpHandler(app *App) http.Handler {
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

		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("error retrieving token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		validID, err := auth.ValidateJWT(token, app.config.JWTSecret)
		if err != nil {
			log.Printf("error validating token: %v", err)
			responseWithError(w, http.StatusUnauthorized, "Something went wrong")
			return
		}

		if params.UserID != validID {
			responseWithError(w, http.StatusForbidden, "Forbidden")
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

		dbChirp, err := app.db.CreateChirp(r.Context(),
			database.CreateChirpParams{
				Body:   str,
				UserID: params.UserID,
			},
		)
		if err != nil {
			log.Printf("error generating chirps: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		createdChirp := newChirpResponse(dbChirp)
		responseWithJSON(w, http.StatusCreated, createdChirp)
	})
}

func updateChirpHandler(app *App) http.Handler {
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

		dbChirp, err := app.db.UpdateChirp(
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

func getChirpsHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbChirps, err := app.db.GetChirps(r.Context())
		if err != nil {
			log.Printf("error retrieving chirps: %v", err)
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

func getChripByIDHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			log.Printf("error parsing chirp id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbChirp, err := app.db.GetChirpByID(r.Context(), chirpID)
		if err != nil {
			log.Printf("error retrieving chirp: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		fetchedChrip := newChirpResponse(dbChirp)
		responseWithJSON(w, http.StatusOK, fetchedChrip)
	})
}

func deleteChirpByID(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := uuid.Parse(r.PathValue("chirpID"))
		if err != nil {
			log.Printf("error parsing user id: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("error retrieving token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		validID, err := auth.ValidateJWT(token, app.config.JWTSecret)
		if err != nil {
			log.Printf("error validating token: %v", err)
			responseWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		dbChirp, err := app.db.GetChirpByID(r.Context(), chirpID)
		if err != nil {
			log.Printf("error retrieving user: %v", err)
			responseWithError(w, http.StatusNotFound, "Not found")
			return
		}

		if dbChirp.UserID != validID {
			responseWithError(w, http.StatusForbidden, "Forbidden")
			return
		}

		if _, err = app.db.DeleteChirpByID(r.Context(), dbChirp.ID); err != nil {
			log.Printf("error deleting user: %v", err)
			responseWithError(w, http.StatusNotFound, "Not found")
			return
		}

		responseWithNoContent(w, http.StatusNoContent)
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
		Error: msg,
	})
}

func responseWithValidationError(w http.ResponseWriter, code int,
	msg string, errors map[string]string) {
	responseWithJSON(w, code, struct {
		Message string            `json:"message"`
		Errors  map[string]string `json:"errors"`
	}{
		Message: msg,
		Errors:  errors,
	})
}

func responseWithNoContent(w http.ResponseWriter, code int) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
}
