package main

import (
	"context"
	"database/sql"
	"os"
	"time"

	"github.com/AlexG695/geo-engine-core/internal/ws"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/AlexG695/geo-engine-core/internal/database"
	"github.com/AlexG695/geo-engine-core/internal/handlers"
	"github.com/gin-contrib/cors"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	sugar := logger.Sugar()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://geo:secretpassword@db:5432/geoengine?sslmode=disable"
	}

	conn, err := sql.Open("pgx", dbURL)
	if err != nil {
		sugar.Fatal("No se pudo conectar a Postgres:", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		sugar.Fatal("BD no responde:", err)
	}
	sugar.Info("✅ Conectado a Postgres + PostGIS")

	redisClient := redis.NewClient(&redis.Options{
		Addr: "geo-redis:6379",
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		sugar.Warn("⚠️ Redis no responde (¿Está corriendo Docker?):", err)
	} else {
		sugar.Info("🚀 Conectado a Redis")
	}

	wsHub := ws.NewHub()

	go wsHub.Run()

	queries := database.New(conn)
	locationHandler := handlers.NewLocationHandler(queries, redisClient, sugar, wsHub)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	locationHandler.RegisterRoutes(r)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "online", "version": "1.0.1-clean"})
	})

	r.GET("/ws", locationHandler.ServeWS)

	sugar.Info("🚀 Geo-Engine iniciando en puerto 8080")
	r.Run(":8080")
}
