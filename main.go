package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func replaceBadWords(s string) string {
	wordList := []string{"kerfuffle", "sharbert", "fornax"}
	tokens := strings.Split(s, " ")
	for idx, token := range tokens {
		for _, word := range wordList {
			if word == strings.ToLower(token) {
				tokens[idx] = "****"
			}
		}
	}
	result := strings.Join(tokens, " ")
	return result
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorChirp struct {
		Error string `json:"error"`
	}
	errValue := errorChirp{
		Error: msg,
	}
	dat, err := json.Marshal(errValue)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	type cleanedSuccess struct {
		CleanedBody string `json:"cleaned_body"`
	}
	resp := cleanedSuccess{
		CleanedBody: payload.(string),
	}

	dat, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(dat)
}

func validationHandler(w http.ResponseWriter, req *http.Request) {
	chirpLength := 140
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(params.Body) > chirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	result := replaceBadWords(params.Body)
	respondWithJSON(w, http.StatusOK, result)
}

func main() {
	rootPath := "."
	port := "8080"
	apiCfg := apiConfig{}
	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", http.StripPrefix("/app",
		apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(rootPath)))))
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.hitsHandler)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	serveMux.HandleFunc("GET /api/healthz", healthzHandler)
	serveMux.HandleFunc("POST /api/validate_chirp", validationHandler)
	server := http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}
	server.ListenAndServe()

}

func healthzHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) hitsHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	content := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	w.Write([]byte(content))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}
