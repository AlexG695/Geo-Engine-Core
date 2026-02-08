package handlers_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // Driver de Postgres
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AlexG695/geo-engine-core/internal/database"
	// Ajusta este import según tu nombre de módulo real si es diferente
	// "github.com/AlexG695/geo-engine-core/internal/handlers"
)

// AJUSTA ESTO: Debe coincidir con tu docker-compose o entorno local
const testDbConn = "postgres://geo:secretpassword@localhost:5432/geoengine?sslmode=disable"

func setupTestDB(t *testing.T) (*sql.DB, *database.Queries) {
	db, err := sql.Open("postgres", testDbConn)
	require.NoError(t, err)

	err = db.Ping()
	require.NoError(t, err, "No se pudo conectar a la BD. ¿Está corriendo Docker?")

	return db, database.New(db)
}

func TestGeofenceLifecycle(t *testing.T) {
	db, queries := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// 1. PREPARACIÓN: Crear un Polígono de Prueba
	// Un cuadrado pequeño cerca de CDMX
	name := "Test Zone " + uuid.New().String()
	geojson := `{
		"type": "Polygon",
		"coordinates": [[
			[-99.168, 19.426],
			[-99.166, 19.426],
			[-99.166, 19.428],
			[-99.168, 19.428],
			[-99.168, 19.426]
		]]
	}`

	// Crear zona
	geofence, err := queries.CreateGeofence(ctx, database.CreateGeofenceParams{
		Name:              name,
		StGeomfromgeojson: geojson,
	})
	require.NoError(t, err)
	t.Logf("✅ Zona creada: %s", geofence.Name)

	// Limpieza al final
	defer func() {
		_ = queries.DeleteGeofence(ctx, geofence.ID)
	}()

	// 2. TEST: Punto DENTRO (-99.167, 19.427 está al centro)
	zonesInside, err := queries.FindGeofencesContainingPoint(ctx, database.FindGeofencesContainingPointParams{
		StMakepoint:   -99.167,
		StMakepoint_2: 19.427,
	})
	require.NoError(t, err)

	found := false
	for _, z := range zonesInside {
		if z.ID == geofence.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "El punto debería estar DENTRO de la zona")

	// 3. TEST: Punto FUERA (0, 0)
	zonesOutside, err := queries.FindGeofencesContainingPoint(ctx, database.FindGeofencesContainingPointParams{
		StMakepoint:   0,
		StMakepoint_2: 0,
	})
	require.NoError(t, err)

	foundOutside := false
	for _, z := range zonesOutside {
		if z.ID == geofence.ID {
			foundOutside = true
			break
		}
	}
	assert.False(t, foundOutside, "El punto (0,0) debería estar FUERA")
}

func TestConcurrentEvents(t *testing.T) {
	// Prueba de estrés: 20 goroutines escribiendo logs al mismo tiempo
	db, queries := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// 1. Necesitamos una zona válida para la Foreign Key
	geoName := "Concurrent Test Zone"
	geoJson := `{"type": "Polygon", "coordinates": [[[-100, 20], [-100.1, 20], [-100.1, 20.1], [-100, 20.1], [-100, 20]]]}`

	zone, err := queries.CreateGeofence(ctx, database.CreateGeofenceParams{
		Name:              geoName,
		StGeomfromgeojson: geoJson,
	})
	require.NoError(t, err)

	// Limpieza
	defer queries.DeleteGeofence(ctx, zone.ID)

	// 2. Ejecutar concurrencia
	var wg sync.WaitGroup
	workers := 20 // Simulamos 20 eventos simultáneos
	deviceID := "test-device-concurrent"

	t.Logf("🚀 Iniciando %d escrituras concurrentes...", workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Insertamos en el historial
			err := queries.LogGeofenceEvent(ctx, database.LogGeofenceEventParams{
				GeofenceID: zone.ID,
				DeviceID:   deviceID,
				EventType:  "ENTER",
			})
			assert.NoError(t, err, "Fallo en escritura concurrente")
		}()
	}

	wg.Wait()
	t.Log("✅ Todas las goroutines terminaron sin errores")
}
