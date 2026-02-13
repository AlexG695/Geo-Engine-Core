package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/AlexG695/geo-engine-core/config"
	"github.com/AlexG695/geo-engine-core/internal/database"
	"github.com/AlexG695/geo-engine-core/internal/handlers"
	"github.com/AlexG695/geo-engine-core/internal/middleware"
	"github.com/AlexG695/geo-engine-core/internal/platform/logger"
	"github.com/AlexG695/geo-engine-core/internal/ws"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	zapLog, err := logger.NewLogger(cfg.EnvMode)
	if err != nil {
		panic("No se pudo configurar el logger: " + err.Error())
	}
	defer zapLog.Sync()
	sugar := zapLog.Sugar()

	r := gin.New()
	r.Use(gin.Recovery())

	conn, err := sql.Open("pgx", cfg.DBConnection)
	if err != nil {
		sugar.Fatal("No se pudo conectar a Postgres:", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		sugar.Fatal("BD no responde:", err)
	}
	sugar.Info("Conectado a Postgres + PostGIS")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		sugar.Warn("Redis no responde (¿Está corriendo Docker?):", err)
	} else {
		sugar.Info("Conectado a Redis")
	}

	wsHub := ws.NewHub()

	go wsHub.Run()

	queries := database.New(conn)
	r = gin.Default()
	r.Use(middleware.RequestLogger(zapLog))

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AllowedOrigins},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Geo-Key", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Geo-Key", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.SetTrustedProxies(nil)
	r.Use(middleware.IPFilter(redisClient))
	r.Use(middleware.RateLimit(redisClient, "100-M"))
	r.Use(middleware.APIKeyAuth(cfg.APISecret))

	locationHandler := handlers.NewLocationHandler(queries, redisClient, sugar, wsHub)
	locationHandler.RegisterRoutes(r)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "online", "version": "1.0.0"})
	})

	r.GET("/ws", locationHandler.ServeWS)

	sugar.Info("Geo-Engine iniciando en puerto 8080")
	r.Run(":8080")
}
