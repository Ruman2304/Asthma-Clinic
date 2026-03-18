package configuration

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var DB *pgxpool.Pool

func InitDB() {
	// Loads .env from the project root
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	// Verify credentials/connection now (so it fails early with clear error)
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Fatal("Failed to ping DB:", err)
	}

	DB = pool
	log.Println("✅ Postgres Connected")
}
