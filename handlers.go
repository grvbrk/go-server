package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (cfg *apiConfig) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) Admin_GetNumberOfHitsHandler(w http.ResponseWriter, r *http.Request) {
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

func (cfg *apiConfig) Admin_ResetNumberOfHitsHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := cfg.db.DeleteAllUsers(r.Context())
	if err != nil {
		log.Printf("Error Deleting all users: %s", err)
		w.WriteHeader(500)
		return
	}

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

func (cfg *apiConfig) AddUserHandler(w http.ResponseWriter, r *http.Request) {
	type reqBodyStruct struct {
		Email string `json:"email"`
	}

	type resBodyStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
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

	user, err := cfg.db.CreateUser(r.Context(), body.Email)
	if err != nil {
		log.Printf("Error creating user: %s", err)
		w.WriteHeader(500)
		return
	}

	resData := resBodyStruct{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	resDataJSON, err := json.Marshal(resData)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(resDataJSON)

}
