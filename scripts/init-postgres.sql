CREATE TABLE IF NOT EXISTS rfid_events (
    id SERIAL PRIMARY KEY,
    tag_id TEXT NOT NULL,
    device_id TEXT,
    gate_id TEXT,
    zone_id TEXT,
    antenna_id TEXT,
    event_type TEXT,
    event_time TIMESTAMPTZ NOT NULL,
    signal_strength DOUBLE PRECISION,
    confidence DOUBLE PRECISION
);

CREATE TABLE IF NOT EXISTS cctv_events (
    id SERIAL PRIMARY KEY,
    camera_id TEXT,
    channel_id TEXT,
    track_id TEXT,
    target_type TEXT,
    object_id TEXT,
    zone_id TEXT,
    direction TEXT,
    event_time TIMESTAMPTZ NOT NULL,
    confidence DOUBLE PRECISION
);

CREATE TABLE IF NOT EXISTS alert_records (
    id TEXT PRIMARY KEY,
    tag_id TEXT,
    camera_id TEXT,
    severity TEXT,
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rfid_tag_time
    ON rfid_events (tag_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_cctv_track_time
    ON cctv_events (track_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_alert_created_at
    ON alert_records (created_at DESC);
