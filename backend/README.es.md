# 🌍 Geo Engine Core

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)
![PostGIS](https://img.shields.io/badge/PostGIS-Enabled-336791?style=flat&logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-Caching-DC382D?style=flat&logo=redis)
![React](https://img.shields.io/badge/React-Frontend-61DAFB?style=flat&logo=react)
![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?style=flat&logo=docker)
![Coverage](https://img.shields.io/badge/Test_Coverage-High-green)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)

Motor de procesamiento geoespacial de alto rendimiento diseñado para rastreo vehicular en tiempo real, gestión dinámica de geocercas y detección de eventos espaciales complejos.

---

## 🚀 Funcionalidades (v1.0)

### 📍 Geocercado Dinámico e Interactivo

Sistema completo para la gestión de zonas geográficas (CRUD) con una interfaz avanzada:

* **Dibujo Vectorial:** Creación de polígonos irregulares mediante interfaz de mapa interactiva.
* **Edición de Precisión:** Ajuste fino de vértices mediante marcadores arrastrables (draggable markers) para corregir zonas sin borrarlas.
* **Validación Espacial:** Algoritmos en Backend (PostGIS) para asegurar la integridad de las geometrías.

### 📜 Auditoría de Eventos (Event Sourcing)

Más allá de las alertas en vivo, el sistema ahora persiste un historial inmutable:

* **Registro Transaccional:** Cada evento de `ENTRADA` y `SALIDA` se almacena en PostgreSQL con consistencia ACID.
* **Análisis Futuro:** Arquitectura lista para soportar reportes de tiempos de estancia y frecuencia de visitas.

### ⚡ Notificaciones en Tiempo Real

* **WebSockets:** Comunicación bidireccional de baja latencia (<50ms) para alertas inmediatas en el Frontend.
* **Gestión de Estado (Redis):** Prevención de "rebotes" de señal y duplicidad de alertas manteniendo el estado de ubicación en memoria caché.

---

## 🧪 Estrategia de Testing y Calidad

Este proyecto implementa una suite de pruebas robusta para garantizar la fiabilidad en entornos de producción:

### 1. Pruebas de Unidad e Integración (Geospatial)

Validamos la lógica espacial contra la base de datos real (PostGIS) para asegurar que las matemáticas no fallen.

* ✅ Verificación de funciones `ST_Contains` y `ST_Intersects`.
* ✅ Validación de parsing de GeoJSON complejos.

### 2. Pruebas de Concurrencia (Stress Testing)

Simulamos escenarios de alto tráfico para garantizar la estabilidad del servidor.

* ✅ **Goroutines:** Tests que lanzan múltiples hilos de escritura simultánea en el log de auditoría para detectar condiciones de carrera (Race Conditions) y bloqueos de base de datos.

**Ejecutar las pruebas:**

```bash
go test ./... -v

```

---

## 🏗️ Arquitectura del Sistema

El sistema sigue una arquitectura limpia y modular:

1. **Ingesta:** Los dispositivos envían coordenadas (Lat/Lng) vía HTTP/UDP.
2. **Procesamiento:**
* **Golang:** Procesa la señal entrante.
* **PostGIS:** Realiza el cálculo espacial ("¿Está este punto dentro de algún polígono?").


3. **Estado:**
* **Redis:** Compara el estado anterior vs actual para determinar si hubo un cambio (Entrada/Salida).


4. **Persistencia:**
* **PostgreSQL:** Guarda el evento en la tabla `geofence_events` (Historial).


5. **Difusión:**
* **WebSockets:** Notifica al Dashboard (React) instantáneamente.



---

## 🛠️ Stack Tecnológico

* **Lenguaje:** Go (Golang)
* **Base de Datos:** PostgreSQL + PostGIS Extension
* **Caché / PubSub:** Redis
* **Frontend:** React + Leaflet + WebSockets
* **Infraestructura:** Docker & Docker Compose
