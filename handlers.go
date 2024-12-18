package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
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

func (cfg *apiConfig) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	type reqBodyStruct struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type resBodyStruct struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
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
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
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
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		log.Printf("Couldn't validate JWT: %s", err)
		w.WriteHeader(401)
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

func (cfg *apiConfig) GetChirpsInAsc(w http.ResponseWriter, r *http.Request) {

	chirps, err := cfg.db.GetAllChirpsInAsc(r.Context())
	if err != nil {
		log.Printf("Error fetching chirps: %s", err)
		w.WriteHeader(500)
		return
	}

	authorID := uuid.Nil
	authorIDString := r.URL.Query().Get("author_id")
	if authorIDString != "" {
		authorID, err = uuid.Parse(authorIDString)
		if err != nil {
			log.Printf("Invalid author ID: %s", err)
			w.WriteHeader(400)
			return
		}
	}

	sortParam := r.URL.Query().Get("sort")

	// Response initiated ---
	chirpResponses := []Chirp{}
	for _, chirp := range chirps {
		if authorID != uuid.Nil && chirp.UserID != authorID {
			continue
		}
		chirpResponse := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		chirpResponses = append(chirpResponses, chirpResponse)
	}

	sort.Slice(chirpResponses, func(i, j int) bool {
		if sortParam == "desc" {
			return chirpResponses[i].CreatedAt.After(chirpResponses[j].CreatedAt)
		}
		return chirpResponses[i].CreatedAt.Before(chirpResponses[j].CreatedAt)
	})

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
		log.Printf("Error getting chirpID: %s", err)
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
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type resBodyStruct struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
		IsChirpyRed  bool      `json:"is_chirpy_red"`
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

	accessToken, err := auth.MakeJWT(user.ID, cfg.jwt_secret, time.Hour)
	if err != nil {
		log.Printf("Unable to create access token: %s", err)
		w.WriteHeader(500)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Unable to create refresh token: %s", err)
		w.WriteHeader(500)
		return
	}

	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().UTC().Add(time.Hour * 24 * 60),
	})
	if err != nil {
		log.Printf("Couldn't save refresh token: %s", err)
		w.WriteHeader(500)
		return
	}

	// Response initiated ---
	resData := resBodyStruct{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        accessToken,
		RefreshToken: refreshToken,
		IsChirpyRed:  user.IsChirpyRed,
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

func (cfg *apiConfig) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	type resBodyStruct struct {
		Token string `json:"token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find token: %s", err)
		w.WriteHeader(400)
		return
	}

	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("Couldn't get user for refresh token: %s", err)
		w.WriteHeader(401)
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwt_secret,
		time.Hour,
	)
	if err != nil {
		log.Printf("Couldn't validate token: %s", err)
		w.WriteHeader(401)
		return
	}

	// Response initiated ---
	resData := resBodyStruct{
		Token: accessToken,
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

func (cfg *apiConfig) RefreshTokenRevokeHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find token: %s", err)
		w.WriteHeader(400)
		return
	}

	_, err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)

	if err != nil {
		log.Printf("Couldn't revoke session: %s", err)
		w.WriteHeader(500)
		return
	}

	// Response initiated ---
	w.WriteHeader(204)
}

func (cfg *apiConfig) UpdateUserCredsHandler(w http.ResponseWriter, r *http.Request) {

	type reqBodyStruct struct {
		NewEmail    string `json:"email"`
		NewPassword string `json:"password"`
	}

	type resBodyStruct struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find JWT token: %s", err)
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		log.Printf("Couldn't validate JWT: %s", err)
		w.WriteHeader(401)
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

	hashedPassword, err := auth.HashPassword(body.NewPassword)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}

	user, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             userID,
		Email:          body.NewEmail,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}

	// Response initiated ---
	resData := resBodyStruct{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
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

func (cfg *apiConfig) DeleteChirpByIdHandler(w http.ResponseWriter, r *http.Request) {

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find JWT token: %s", err)
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		log.Printf("Couldn't validate JWT: %s", err)
		w.WriteHeader(403)
		return
	}

	chirpId, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		log.Printf("Error getting chirpID: %s", err)
		w.WriteHeader(500)
		return
	}

	chirp, err := cfg.db.GetChirpById(r.Context(), chirpId)
	if err != nil {
		log.Printf("Couldn't find chirp for the given user: %s", err)
		w.WriteHeader(404)
		return
	}

	if chirp.UserID != userID {
		log.Printf("You can't delete this chirp: %s", err)
		w.WriteHeader(403)
		return
	}

	err = cfg.db.DeleteChirpById(r.Context(), chirpId)
	if err != nil {
		log.Printf("Error deleting chirp: %s", err)
		w.WriteHeader(500)
		return
	}

	// Response initiated ---
	w.WriteHeader(204)
}

func (cfg *apiConfig) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	type reqBodyStruct struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		}
	}

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		log.Printf("Couldn't find api key: %s", err)
		w.WriteHeader(401)
		return
	}

	if apiKey != cfg.polka_key {
		log.Printf("API key is invalid: %s", err)
		w.WriteHeader(401)
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

	if body.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	_, err = cfg.db.UpgradeToChirpyRed(r.Context(), body.Data.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("Couldn't find user: %s", err)
			w.WriteHeader(404)
			return
		}
		log.Printf("Couldn't update user: %s", err)
		w.WriteHeader(500)
		return
	}

	// Response initiated ---
	w.WriteHeader(204)
}
