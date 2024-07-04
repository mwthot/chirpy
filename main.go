package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type apiConfig struct {
	fileserverHits int
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	apiCfg := apiConfig{
		fileserverHits: 0,
	}

	mux := http.NewServeMux()
	mux.Handle("/app/*", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("/api/reset", apiCfg.handlerReset)
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.requestValidator)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset = utf-8")
	w.WriteHeader(http.StatusOK)
	htmlTemplate := `
		<html> 
		<body> 
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body> 
		</html>
	`
	w.Write([]byte(fmt.Sprintf(htmlTemplate, cfg.fileserverHits)))
}

func (cfg *apiConfig) requestValidator(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Body string `json:"body"`
	}
	type response struct {
		CleanedBody string `json:"cleaned_body`
	}

	var reqData request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqData)
	if err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(reqData.Body) > 140 {
		http.Error(w, `{"error": Chirp is too long}`, http.StatusBadRequest)
	}

	cleanedBody := cfg.profanityDetector(reqData.Body)

	respData := response{
		CleanedBody: cleanedBody,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(respData); err != nil {
		http.Error(w, `{"error": "Unable to encode response}`, http.StatusInternalServerError)
	}

}

func (cfg *apiConfig) profanityDetector(text string) string {
	profanity := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(text, " ")

	for i, word := range words {
		lowerWord := strings.ToLower(word)
		for _, badWord := range profanity {
			if lowerWord == badWord {
				words[i] = "****"
				break
			}
		}
	}

	cleanedText := strings.Join(words, " ")
	return cleanedText
}
