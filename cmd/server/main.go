package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"microhabits/internal/auth"
	"microhabits/internal/db"
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

	authHandler := auth.NewHandler(auth.NewService(database, jwtSecret))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	log.Println("MicroHabits API listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
