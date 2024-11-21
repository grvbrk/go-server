package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"githuv.com/grvbrk/go-server/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	jwt_secret     string
}

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	jwt_secret := os.Getenv("JWT_SECRET")

	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		fmt.Printf("Error %v", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
		jwt_secret:     jwt_secret,
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", apiCfg.HealthCheckHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.Admin_GetNumberOfHitsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.Admin_ResetNumberOfHitsHandler)
	mux.HandleFunc("POST /api/users", apiCfg.AddUserHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.CreateChirpHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.GetAllChirpsInAsc)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.GetChirpById)
	mux.HandleFunc("POST /api/login", apiCfg.LoginUser)
	mux.HandleFunc("POST /api/refresh", apiCfg.RefreshTokenHandler)
	mux.HandleFunc("POST /api/revoke", apiCfg.RefreshTokenRevokeHandler)

	appServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Printf("Server is starting on port %v \n", appServer.Addr)

	appServer.ListenAndServe()

}
