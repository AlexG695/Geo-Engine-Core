-- name: CreateLocation :one
-- Guarda una nueva ubicación y devuelve el ID insertado.
INSERT INTO locations (
    id, device_id, latitude, longitude, accuracy, heading, speed, is_mock, created_at
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9
         )
    RETURNING id;

-- name: GetLatestLocationByDevice :one
-- Obtiene la última ubicación conocida de un dispositivo.
SELECT * FROM locations
WHERE device_id = $1
ORDER BY created_at DESC
    LIMIT 1;



-- name: GetNearbyDrivers :many
-- Busca conductores dentro de un radio (en metros) usando PostGIS.
-- ST_DWithin usa índices espaciales, así que es ULTRA rápido.
SELECT
    id, device_id, latitude, longitude, heading, speed, created_at
FROM locations
WHERE
  -- Compara la columna geom contra un punto creado al vuelo
    ST_DWithin(
            geom,
            ST_SetSRID(ST_MakePoint(@lng::float8, @lat::float8), 4326)::geography,
            @radius_meters::float8
    )
  AND created_at > NOW() - INTERVAL '5 minutes' -- Solo conductores activos recientemente
ORDER BY created_at DESC;



-- name: GetDriverRoute :one
SELECT
    COALESCE(
            ST_AsGeoJSON(ST_MakeLine(geom ORDER BY created_at))::text,
            '{"type": "LineString", "coordinates": []}'
    )::text as geojson_route
FROM locations
WHERE
    device_id = $1;


-- name: GetGeofences :many
SELECT id, name, ST_AsGeoJSON(area)::text as geojson
FROM geofences;

-- name: FindGeofencesContainingPoint :many
SELECT id, name
FROM geofences
WHERE ST_Contains(area, ST_SetSRID(ST_MakePoint($1, $2), 4326));


-- name: CreateGeofence :one
INSERT INTO geofences (name, area)
VALUES ($1, ST_GeomFromGeoJSON($2)) -- <-- Recibe un string GeoJSON
    RETURNING id, name;

-- name: DeleteGeofence :exec
DELETE FROM geofences WHERE id = $1;

-- name: UpdateGeofence :one
UPDATE geofences
SET
    name = $2,
    area = CASE
               WHEN length($3::text) > 0 THEN ST_GeomFromGeoJSON($3)
               ELSE area
        END
WHERE id = $1
    RETURNING id, name, ST_AsGeoJSON(area)::text as geojson;

-- name: LogGeofenceEvent :exec
INSERT INTO geofence_events (geofence_id, device_id, event_type)
VALUES ($1, $2, $3);