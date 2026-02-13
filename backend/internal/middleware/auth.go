package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientKey := c.GetHeader("X-Geo-Key")

		if clientKey == "" {
			clientKey = c.Query("key")
		}

		if clientKey == "" || clientKey != secret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid API Key"})
			return
		}

		c.Next()
	}
}
