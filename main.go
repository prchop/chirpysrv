package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

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
		log.Print(err)
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
		log.Print(err)
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
	cfg := NewAPIConfig()
	mux := http.NewServeMux()

	mw := func(h http.Handler) http.Handler {
		return cfg.MiddlewareMetricsInc(h)
	}

	// User endpoint
	mux.Handle("/app/", mw(appHandler(rootFilepath)))

	// API endpoint
	mux.Handle("GET /api/health", mw(http.HandlerFunc(healthHandler)))
	mux.Handle("POST /api/validate_chirp", mw(http.HandlerFunc(validateHandler)))

	// Admin endpoint
	mux.Handle("GET /admin/metrics", cfg.HandlerMetrics())
	mux.Handle("POST /admin/reset", cfg.HandlerReset())

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	log.Printf("Chripy server start at http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
