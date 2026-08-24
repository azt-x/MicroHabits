package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"microhabits/internal/auth"
	"microhabits/internal/db"
	"microhabits/internal/habits"
)

func main() {
	config, err := godotenv.Read(".env")
	if err != nil {
		log.Fatalf("read .env: %v", err)
	}

	databasePath := config["DATABASE_PATH"]
	if databasePath == "" {
		databasePath = "./microhabits.db"
	}
	jwtSecret := config["JWT_SECRET"]
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	database, err := db.Open(context.Background(), databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	authService := auth.NewService(database, jwtSecret)
	authHandler := auth.NewHandler(authService)
	habitService := habits.NewService(database)
	habitHandler := habits.NewHandler(habitService, authService)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("GET /me", authHandler.Me)
	mux.HandleFunc("GET /habits", habitHandler.ListHabits)
	mux.HandleFunc("POST /habits", habitHandler.CreateHabit)
	mux.HandleFunc("GET /habits/{id}", habitHandler.GetHabit)
	mux.HandleFunc("PUT /habits/{id}", habitHandler.UpdateHabit)
	mux.HandleFunc("DELETE /habits/{id}", habitHandler.DeleteHabit)
	mux.HandleFunc("GET /habits/{id}/completed", habitHandler.ListCompletions)
	mux.HandleFunc("POST /habits/{id}/completed", habitHandler.CreateCompletion)
	mux.HandleFunc("DELETE /habits/{id}/completed/{completionId}", habitHandler.DeleteCompletion)

	log.Println("MicroHabits API listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func health(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
