CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_raster;
CREATE EXTENSION IF NOT EXISTS h3;
CREATE EXTENSION IF NOT EXISTS h3_postgis;

CREATE TABLE locations (
                           id UUID PRIMARY KEY,
                           device_id VARCHAR(255) NOT NULL,
                           latitude DOUBLE PRECISION NOT NULL,
                           longitude DOUBLE PRECISION NOT NULL,
                           geom GEOMETRY(Point, 4326) GENERATED ALWAYS AS (ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)) STORED,
                           h3_ix h3index GENERATED ALWAYS AS (h3_lat_lng_to_cell(ST_MakePoint(longitude, latitude), 9)) STORED,
                           accuracy DOUBLE PRECISION,
                           heading DOUBLE PRECISION,
                           speed DOUBLE PRECISION,
                           is_mock BOOLEAN DEFAULT FALSE,
                           created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE geofences (
                           id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                           name VARCHAR(255) NOT NULL,
                           area GEOMETRY(Polygon, 4326) NOT NULL,
                           created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


CREATE TABLE geofence_events (
                                 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                 geofence_id UUID NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
                                 device_id VARCHAR(50) NOT NULL,
                                 event_type VARCHAR(10) NOT NULL,
                                 timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);



CREATE INDEX idx_events_device ON geofence_events(device_id);
CREATE INDEX idx_events_geofence ON geofence_events(geofence_id);
CREATE INDEX idx_geofences_area ON geofences USING GIST (area);
CREATE INDEX idx_locations_geom ON locations USING GIST (geom);
CREATE INDEX idx_locations_device_time ON locations (device_id, created_at DESC);
CREATE INDEX idx_locations_h3 ON locations (h3_ix);