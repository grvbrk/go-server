package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"githuv.com/grvbrk/go-server/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		fmt.Printf("Error %v", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiCfg := apiConfig{
		dbQueries: dbQueries,
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.GetNumberOfHitsHandlerAdmin)
	mux.HandleFunc("POST /admin/reset", apiCfg.ResetNumberOfHitsHandlerAdmin)
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.ValidateChirp)

	appServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Printf("Server is starting on port %v \n", appServer.Addr)

	appServer.ListenAndServe()

}

func (cfg *apiConfig) GetNumberOfHitsHandlerAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	text := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	w.Write([]byte(text))
}

func (cfg *apiConfig) ResetNumberOfHitsHandlerAdmin(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
}

func (cfg *apiConfig) ValidateChirp(w http.ResponseWriter, r *http.Request) {
	type reqBodyStruct struct {
		Body string `json:"body"`
	}

	type resBodyStruct struct {
		CleanedBody string `json:"cleaned_body"`
	}

	type resErrorStruct struct {
		Error string `json:"error"`
	}

	body := reqBodyStruct{}
	reqBodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	err = json.Unmarshal(reqBodyBytes, &body)
	if err != nil {
		log.Printf("Error unmarshalling body: %s", err)
		w.WriteHeader(500)
		return
	}

	if len(body.Body) > 140 {
		resErrorData := resErrorStruct{
			Error: "Chirp is too long",
		}

		resErrorJSON, err := json.Marshal(resErrorData)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write(resErrorJSON)
		return
	}

	wordsToHide := []string{"kerfuffle", "sharbert", "fornax"}
	pattern := regexp.MustCompile(`(?i)\b(` + strings.Join(wordsToHide, "|") + `)\b`)
	result := pattern.ReplaceAllString(body.Body, "****")

	resData := resBodyStruct{
		CleanedBody: result,
	}

	resDataJSON, err := json.Marshal(resData)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(resDataJSON)

}
