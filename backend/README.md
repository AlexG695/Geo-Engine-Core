# 🌍 Geo Engine Core

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)
![PostGIS](https://img.shields.io/badge/PostGIS-Enabled-336791?style=flat&logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-Caching-DC382D?style=flat&logo=redis)
![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?style=flat&logo=docker)
![Coverage](https://img.shields.io/badge/Test_Coverage-High-green)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)


High-performance geospatial processing engine designed for real-time vehicle tracking, dynamic geofencing management, and complex spatial event detection.

## 🚀 Key Features (v1.0)

### 📍 Dynamic & Interactive Geofencing

A complete system for managing geographic zones (CRUD) with an advanced interface:

* **Vector Drawing:** Creation of irregular polygons via an interactive map interface.
* **Precision Editing:** Fine-tuning of vertices using **draggable markers** to correct zones without deleting them.
* **Spatial Validation:** Backend algorithms (PostGIS) ensure geometric integrity and prevent invalid shapes.

### 📜 Event Sourcing & Audit Log

Beyond live alerts, the system now persists an immutable history:

* **Transactional Logging:** Every `ENTER` and `EXIT` event is stored in PostgreSQL with ACID consistency.
* **Future-Proof Analytics:** Architecture ready to support dwell-time reports and visit frequency analysis.

### ⚡ Real-Time Notifications

* **WebSockets:** Low-latency (<50ms) bidirectional communication for immediate frontend alerts.
* **State Management (Redis):** Prevents signal bouncing and duplicate alerts by maintaining location state in an in-memory cache.

## 🧪 Quality Assurance & Testing Strategy

This project implements a robust testing suite to ensure reliability in production environments:

### 1. Unit & Integration Testing (Geospatial)

Validates spatial logic against the real database (PostGIS) to ensure math accuracy.

* ✅ Verification of `ST_Contains` and `ST_Intersects` functions.
* ✅ Validation of complex GeoJSON parsing.

### 2. Concurrency Stress Testing

Simulates high-traffic scenarios to ensure server stability.

* ✅ **Goroutines:** Tests that launch multiple simultaneous write threads to the audit log to detect **Race Conditions** and database locking issues.

**Run tests:**

```bash
go test ./... -v

```

## 🏗️ System Architecture

The system follows a clean and modular architecture:

1. **Ingestion:** Devices send coordinates (Lat/Lng) via HTTP/UDP.
2. **Processing:**
* **Golang:** Processes the incoming signal.
* **PostGIS:** Performs spatial calculation ("Is this point inside any polygon?").


3. **State:**
* **Redis:** Compares previous vs. current state to determine if a change (Entry/Exit) occurred.


4. **Persistence:**
* **PostgreSQL:** Stores the event in the `geofence_events` table (History).


5. **Broadcast:**
* **WebSockets:** Notifies the Dashboard (React) instantly.

---

## 🛠️ Technology Stack

* **Language:** Go (Golang)
* **Database:** PostgreSQL + PostGIS Extension
* **Cache / PubSub:** Redis
* **Frontend:** React + Leaflet + WebSockets
* **Infrastructure:** Docker & Docker Compose