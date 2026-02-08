# 💻 Geo Engine - Frontend Dashboard

![React](https://img.shields.io/badge/React-Frontend-61DAFB?style=flat&logo=react)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)


A modern real-time dashboard built with **React**, **TypeScript**, and **Vite**. It provides a visual interface for tracking vehicles and managing geofences with precision.

## 🛠️ Tech Stack

* **Framework:** React 18 + TypeScript
* **Build Tool:** Vite
* **Maps:** React Leaflet + Leaflet API
* **Styling:** CSS Modules / Tailwind (optional)
* **Testing:** Vitest + React Testing Library

## 🚀 Local Development

1.  **Install Dependencies:**
    ```bash
    npm install
    ```

2.  **Start Dev Server:**
    ```bash
    npm run dev
    ```
    The app will be available at `http://localhost:5173`.

## 📍 Key Components

### `MapContainer` & `DrawLogic`
We implemented a custom drawing engine instead of using standard drawing libraries to have full control over the UX:
* **Draggable Markers:** When editing a polygon, vertices become draggable markers (`L.Marker` with `draggable={true}`).
* **Visual Feedback:** Dashed lines appear while drawing to indicate the closing path.

### `WebSocketContext`
Manages the global connection to the backend. It handles:
* Reconnection logic.
* Dispatching `LOCATION_UPDATE` events to the map.
* Dispatching `GEOFENCE_EVENT` alerts (Toasts).

## 🧪 Testing

We use **Vitest** for unit and component testing. We mock `react-leaflet` and `axios` to isolate UI logic.

```bash
# Run UI tests
npm test

```

### Test Coverage

* Map rendering and initialization.
* Data fetching mocks (Drivers & Geofences).
* Drawing mode state transitions.
* Validation logic (e.g., preventing saving polygons with < 3 points).