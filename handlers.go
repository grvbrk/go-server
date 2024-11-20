package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"githuv.com/grvbrk/go-server/internal/auth"
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
		Email    string `json:"email"`
		Password string `json:"password"`
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

	hashedPassword, err := auth.HashPassword(body.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          body.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		log.Printf("Error creating user: %s", err)
		w.WriteHeader(500)
		return
	}

	// Response initiated ---
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
		Body string `json:"body"`
	}

	type resBodyStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	// Check for access token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting bearer/access token: %s", err)
		w.WriteHeader(500)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		log.Printf("Couldn't validate JWT: %s", err)
		w.WriteHeader(500)
		return
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
		UserID: userID,
		Body:   body.Body,
	})
	if err != nil {
		log.Printf("Error unmarshalling body: %s", err)
		w.WriteHeader(500)
		return
	}

	// Response initiated ---
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

func (cfg *apiConfig) LoginUser(w http.ResponseWriter, r *http.Request) {
	type reqBodyStruct struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	type resBodyStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Token     string    `json:"token"`
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

	user, err := cfg.db.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		log.Printf("Error finding user: %s", err)
		w.WriteHeader(500)
		return
	}

	err = auth.CheckPasswordHash(body.Password, user.HashedPassword)
	if err != nil {
		log.Printf("Unauthenticated User: %s", err)
		w.WriteHeader(401)
		return
	}

	expirationTime := time.Hour
	if body.ExpiresInSeconds > 0 && body.ExpiresInSeconds < 3600 {
		expirationTime = time.Duration(body.ExpiresInSeconds) * time.Second
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.jwt_secret, expirationTime)
	if err != nil {
		log.Printf("Unable to create access token: %s", err)
		w.WriteHeader(401)
		return
	}

	// Response initiated ---
	resData := resBodyStruct{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     accessToken,
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
