package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const LoggerKey = "zapLogger"

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		requestLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("query", query),
		)

		c.Set(LoggerKey, requestLogger)
		c.Header("X-Request-ID", requestID)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		errorMsg := c.Errors.ByType(gin.ErrorTypePrivate).String()

		logFields := []zapcore.Field{
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
		}

		if len(errorMsg) > 0 {
			logFields = append(logFields, zap.String("error", errorMsg))
		}

		msg := "Request Completed"
		if statusCode >= 500 {
			requestLogger.Error(msg, logFields...)
		} else if statusCode >= 400 {
			requestLogger.Warn(msg, logFields...)
		} else {
			requestLogger.Info(msg, logFields...)
		}
	}
}

func GetLogger(c *gin.Context) *zap.Logger {
	if logger, exists := c.Get(LoggerKey); exists {
		return logger.(*zap.Logger)
	}
	return zap.L()
}
