package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"githuv.com/grvbrk/go-server/internal/database"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

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

func (cfg *apiConfig) CreateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type reqBodyStruct struct {
		UserId uuid.UUID `json:"user_id"`
		Body   string    `json:"body"`
	}

	type resBodyStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
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

	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		UserID: body.UserId,
		Body:   body.Body,
	})
	if err != nil {
		log.Printf("Error unmarshalling body: %s", err)
		w.WriteHeader(500)
		return
	}

	resData := resBodyStruct{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
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

func (cfg *apiConfig) GetAllChirpsInAsc(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetAllChirpsInAsc(r.Context())
	if err != nil {
		log.Printf("Error fetching chirps: %s", err)
		w.WriteHeader(500)
		return
	}

	var chirpResponses []Chirp
	for _, chirp := range chirps {
		chirpResponse := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		chirpResponses = append(chirpResponses, chirpResponse)
	}

	resDataJSON, err := json.Marshal(chirpResponses)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(resDataJSON)
}

func (cfg *apiConfig) GetChirpById(w http.ResponseWriter, r *http.Request) {
	chirpId, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		log.Printf("Error fetching chirps: %s", err)
		w.WriteHeader(500)
		return
	}

	chirp, err := cfg.db.GetChirpById(r.Context(), chirpId)
	if err != nil {
		log.Printf("Error fetching chirps: %s", err)
		w.WriteHeader(404)
		return
	}

	resData := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
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
