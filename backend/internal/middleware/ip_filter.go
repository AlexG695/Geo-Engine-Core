package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const BlacklistKey = "firewall:blocked_ips"

func IPFilter(client *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		isBlocked, err := client.SIsMember(c.Request.Context(), BlacklistKey, clientIP).Result()

		if err != nil {
			c.Next()
			return
		}

		if isBlocked {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access Denied: Your IP is on the blacklist.",
			})
			return
		}

		c.Next()
	}
}
