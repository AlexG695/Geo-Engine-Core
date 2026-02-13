package middleware

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimit(client *redis.Client, rateStr string) gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted(rateStr)
	if err != nil {
		log.Fatal(err)
	}

	store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix: "limiter",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 3. Crear el middleware
	middleware := mgin.NewMiddleware(limiter.New(store, rate))

	return middleware
}
