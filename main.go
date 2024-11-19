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
}

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		fmt.Printf("Error %v", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiCfg := apiConfig{
		db:       dbQueries,
		platform: platform,
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", apiCfg.HealthCheckHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.Admin_GetNumberOfHitsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.Admin_ResetNumberOfHitsHandler)
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.ValidateChirp)
	mux.HandleFunc("POST /api/users", apiCfg.AddUserHandler)

	appServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Printf("Server is starting on port %v \n", appServer.Addr)

	appServer.ListenAndServe()

}
