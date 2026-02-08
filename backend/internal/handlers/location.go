package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AlexG695/geo-engine-core/internal/database"
	"github.com/AlexG695/geo-engine-core/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type LocationHandler struct {
	queries     *database.Queries
	redisClient *redis.Client
	logger      *zap.SugaredLogger
	hub         *ws.Hub
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type CreateGeofenceRequest struct {
	Name    string `json:"name" binding:"required"`
	GeoJSON string `json:"geojson" binding:"required"`
}

func NewLocationHandler(q *database.Queries, r *redis.Client, l *zap.SugaredLogger, h *ws.Hub) *LocationHandler {
	return &LocationHandler{
		queries:     q,
		redisClient: r,
		logger:      l,
		hub:         h,
	}
}

func (h *LocationHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/location", h.CreateLocation)
	r.GET("/drivers/nearby", h.GetNearbyDrivers)
	r.GET("/drivers/:id/route", h.GetDriverRoute)
	r.GET("/geofences", h.GetGeofences)
	r.POST("/geofences", h.CreateGeofence)
	r.DELETE("/geofences/:id", h.DeleteGeofence)
	r.PUT("/geofences/:id", h.UpdateGeofence)
}

func (h *LocationHandler) CreateLocation(c *gin.Context) {
	var req struct {
		DeviceID  string  `json:"device_id" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
		Speed     float64 `json:"speed"`
		Heading   float64 `json:"heading"`
		Accuracy  float64 `json:"accuracy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coordenadas fuera de rango"})
		return
	}

	go h.checkGeofences(req.DeviceID, req.Latitude, req.Longitude)

	id, _ := uuid.NewV7()

	insertedID, err := h.queries.CreateLocation(c, database.CreateLocationParams{
		ID:        id,
		DeviceID:  req.DeviceID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Speed:     sql.NullFloat64{Float64: req.Speed, Valid: true},
		Heading:   sql.NullFloat64{Float64: req.Heading, Valid: true},
		Accuracy:  sql.NullFloat64{Float64: req.Accuracy, Valid: true},
		IsMock:    sql.NullBool{Bool: false, Valid: true},
	})

	if err != nil {
		h.logger.Errorw("Error guardando ubicación", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno"})
		return
	}

	errRedis := h.redisClient.GeoAdd(c, "drivers:locations", &redis.GeoLocation{
		Name:      req.DeviceID,
		Longitude: req.Longitude,
		Latitude:  req.Latitude,
	}).Err()

	if errRedis != nil {
		h.logger.Warnw("⚠️ Falló actualización en Redis", "error", errRedis)
	}

	updatePayload := gin.H{
		"type":      "LOCATION_UPDATE",
		"device_id": req.DeviceID,
		"latitude":  req.Latitude,
		"longitude": req.Longitude,
		"heading":   req.Heading,
	}

	h.hub.SendUpdate(updatePayload)
	c.JSON(201, gin.H{"status": "created", "id": insertedID})
}

func (h *LocationHandler) GetNearbyDrivers(c *gin.Context) {
	var params struct {
		Lat    float64 `form:"lat" binding:"required"`
		Lng    float64 `form:"lng" binding:"required"`
		Radius float64 `form:"radius" binding:"required"`
	}

	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Faltan coordenadas o radio"})
		return
	}

	locations, err := h.redisClient.GeoSearchLocation(c, "drivers:locations",
		&redis.GeoSearchLocationQuery{
			GeoSearchQuery: redis.GeoSearchQuery{
				Longitude:  params.Lng,
				Latitude:   params.Lat,
				Radius:     params.Radius,
				RadiusUnit: "m",
			},
			WithCoord: true,
			WithDist:  true,
		},
	).Result()

	if err == nil && len(locations) > 0 {
		var response []gin.H
		for _, loc := range locations {
			response = append(response, gin.H{
				"device_id": loc.Name,
				"latitude":  loc.Latitude,
				"longitude": loc.Longitude,
				"distance":  loc.Dist,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"source": "redis-cache",
			"count":  len(response),
			"data":   response,
		})
		return
	}

	drivers, err := h.queries.GetNearbyDrivers(c, database.GetNearbyDriversParams{
		Lng:          params.Lng,
		Lat:          params.Lat,
		RadiusMeters: params.Radius,
	})

	if err != nil {
		h.logger.Errorw("Error buscando cercanos", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en el radar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": len(drivers), "data": drivers, "source": "database"})
}

func (h *LocationHandler) GetDriverRoute(c *gin.Context) {
	deviceID := c.Param("id")

	routeJSON, err := h.queries.GetDriverRoute(c, deviceID)
	if err != nil {
		h.logger.Errorw("Error obteniendo ruta", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo calcular la ruta"})
		return
	}

	c.Data(http.StatusOK, "application/json", []byte(routeJSON))
}

func (h *LocationHandler) ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Falló upgrade WS:", err)
		return
	}

	h.hub.Register <- conn

	go func() {
		defer func() { h.hub.Unregister <- conn }()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (h *LocationHandler) sendGeofenceEvent(deviceID, zoneName, eventType string) {
	event := gin.H{
		"type":      "GEOFENCE_EVENT",
		"device_id": deviceID,
		"zone_name": zoneName,
		"event":     eventType,
		"timestamp": time.Now(),
	}
	h.hub.SendUpdate(event)
	h.logger.Infow("🚨 GEOFENCE CHANGE", "device", deviceID, "event", eventType, "zone", zoneName)
}

func (h *LocationHandler) checkGeofences(deviceID string, lat, lng float64) {
	ctx := context.Background()

	// 1. Obtener zonas ACTUALES (PostGIS)
	currentZones, err := h.queries.FindGeofencesContainingPoint(ctx, database.FindGeofencesContainingPointParams{
		StMakepoint:   lng,
		StMakepoint_2: lat,
	})
	if err != nil {
		h.logger.Error("Error checking geofences", err)
		return
	}

	currentZoneSet := make(map[string]bool)
	for _, z := range currentZones {
		key := fmt.Sprintf("%s|%s", z.ID.String(), z.Name)
		currentZoneSet[key] = true
	}

	redisKey := fmt.Sprintf("driver:zones:%s", deviceID)
	prevZones, err := h.redisClient.SMembers(ctx, redisKey).Result()
	if err != nil {
		prevZones = []string{}
	}

	for _, prevKey := range prevZones {
		if !currentZoneSet[prevKey] {
			parts := strings.Split(prevKey, "|")
			if len(parts) < 2 {
				continue
			}

			idStr := parts[0]
			name := parts[1]

			zoneID, _ := uuid.Parse(idStr)
			if len(parts) > 1 {
				name = parts[1]
			}

			h.sendGeofenceEvent(deviceID, name, "EXIT")

			go func(zid uuid.UUID, did string) {
				h.queries.LogGeofenceEvent(context.Background(), database.LogGeofenceEventParams{
					GeofenceID: zid,
					DeviceID:   did,
					EventType:  "EXIT",
				})
			}(zoneID, deviceID)
		}
	}

	pipeline := h.redisClient.Pipeline()
	pipeline.Del(ctx, redisKey)

	for currentKey := range currentZoneSet {
		isNew := true
		for _, prevKey := range prevZones {
			if prevKey == currentKey {
				isNew = false
				break
			}
		}

		if isNew {
			parts := strings.Split(currentKey, "|")
			if len(parts) < 2 {
				continue
			}

			idStr := parts[0]
			name := parts[1]

			// 2. Parseamos a UUID
			zoneID, _ := uuid.Parse(idStr)
			h.sendGeofenceEvent(deviceID, name, "ENTER")
			go func(zid uuid.UUID, did string) {
				h.queries.LogGeofenceEvent(context.Background(), database.LogGeofenceEventParams{
					GeofenceID: zid,
					DeviceID:   did,
					EventType:  "ENTER",
				})
			}(zoneID, deviceID)
		}

		pipeline.SAdd(ctx, redisKey, currentKey)
	}

	if len(currentZoneSet) > 0 {
		pipeline.Expire(ctx, redisKey, 24*time.Hour)
	}

	pipeline.Exec(ctx)
}

func (h *LocationHandler) GetGeofences(c *gin.Context) {
	zones, err := h.queries.GetGeofences(c)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error cargando zonas"})
		return
	}
	c.JSON(200, zones)
}

func (h *LocationHandler) CreateGeofence(c *gin.Context) {
	var req CreateGeofenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := h.queries.CreateGeofence(c, database.CreateGeofenceParams{
		Name:              req.Name,
		StGeomfromgeojson: req.GeoJSON,
	})

	if err != nil {
		h.logger.Error("Error creating geofence from drawing", err)
		c.JSON(500, gin.H{"error": "No se pudo guardar la zona dibujada"})
		return
	}

	c.Status(201)
}

func (h *LocationHandler) DeleteGeofence(c *gin.Context) {
	idStr := c.Param("id")

	var id uuid.UUID
	var err error
	if id, err = uuid.Parse(idStr); err != nil {
		c.JSON(400, gin.H{"error": "ID inválido"})
		return
	}

	err = h.queries.DeleteGeofence(c, id)
	if err != nil {
		h.logger.Error("Error deleting geofence", err)
		c.JSON(500, gin.H{"error": "No se pudo eliminar"})
		return
	}

	c.JSON(200, gin.H{"message": "Eliminado"})
}

func (h *LocationHandler) UpdateGeofence(c *gin.Context) {
	idStr := c.Param("id")
	var id uuid.UUID
	var err error
	if id, err = uuid.Parse(idStr); err != nil {
		c.JSON(400, gin.H{"error": "ID inválido"})
		return
	}

	var req struct {
		Name    string `json:"name" binding:"required"`
		GeoJSON string `json:"geojson"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.queries.UpdateGeofence(c, database.UpdateGeofenceParams{
		ID:      id,
		Name:    req.Name,
		Column3: req.GeoJSON,
	})

	if err != nil {
		h.logger.Error("Error updating geofence", err)
		c.JSON(500, gin.H{"error": "No se pudo actualizar"})
		return
	}

	c.JSON(200, updated)
}
