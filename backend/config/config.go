package config

import (
	"log"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	DBConnection string `env:"DBConnection,required"`

	Port string `env:"PORT" envDefault:"8080"`

	APISecret string `env:"API_SECRET,required"`

	RedisAddr string `env:"REDIS_ADDR" envDefault:"redis:6379"`

	RedisPassword string `env:"REDIS_PASSWORD" envDefault:""`

	EnvMode string `env:"ENV_MODE" envDefault:"development"`

	AllowedOrigins string `env:"ALLOWED_ORIGINS" envDefault:"http://localhost:5173"`
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("FATAL: Faltan variables de entorno requeridas:\n%v", err)
	}

	return &cfg
}
