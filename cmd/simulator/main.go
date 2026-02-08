package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Configuración del Simulador
const (
	API_URL     = "http://localhost:8080/location"
	NUM_DRIVERS = 10
	CENTER_LAT  = 28.6353
	CENTER_LNG  = -106.0889
	MOVE_STEP   = 0.0005
)

type Driver struct {
	DeviceID  string  `json:"device_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Heading   float64 `json:"heading"`
}

func main() {
	fmt.Printf("🚀 Iniciando simulación con %d conductores en Chihuahua...\n", NUM_DRIVERS)

	var wg sync.WaitGroup

	for i := 1; i <= NUM_DRIVERS; i++ {
		wg.Add(1)
		driverID := fmt.Sprintf("taxi-sim-%03d", i)

		startLat := CENTER_LAT + (rand.Float64()*0.02 - 0.01)
		startLng := CENTER_LNG + (rand.Float64()*0.02 - 0.01)

		go simulateDriver(driverID, startLat, startLng, &wg)
	}

	wg.Wait()
}

func simulateDriver(id string, lat, lng float64, wg *sync.WaitGroup) {
	defer wg.Done()

	driver := Driver{
		DeviceID:  id,
		Latitude:  lat,
		Longitude: lng,
		Speed:     rand.Float64() * 60,
		Heading:   rand.Float64() * 360,
	}

	client := &http.Client{Timeout: 2 * time.Second}

	for {
		driver.Latitude += (rand.Float64()*MOVE_STEP - (MOVE_STEP / 2))
		driver.Longitude += (rand.Float64()*MOVE_STEP - (MOVE_STEP / 2))

		payload, _ := json.Marshal(driver)
		resp, err := client.Post(API_URL, "application/json", bytes.NewBuffer(payload))

		if err != nil {
			fmt.Printf("❌ [%s] Error enviando: %v\n", id, err)
		} else {
			if resp.StatusCode != 201 {
				fmt.Printf("⚠️ [%s] API rechazó: %d\n", id, resp.StatusCode)
			}
			resp.Body.Close()
		}

		sleepTime := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
		time.Sleep(sleepTime)
	}
}
