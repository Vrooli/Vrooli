-- App Monitor Database Schema

CREATE TABLE IF NOT EXISTS apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    scenario_name VARCHAR(255) NOT NULL,
    path VARCHAR(500) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) DEFAULT 'stopped',
    port_mappings JSONB,
    environment JSONB,
    config JSONB
);

CREATE TABLE IF NOT EXISTS app_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    cpu_usage FLOAT,
    memory_usage FLOAT,
    disk_usage FLOAT,
    network_in BIGINT,
    network_out BIGINT,
    container_id VARCHAR(255),
    pid INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS app_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    metric_type VARCHAR(100) NOT NULL,
    value FLOAT NOT NULL,
    metadata JSONB,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS app_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL,
    message TEXT,
    source VARCHAR(100),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_app_status_app_id ON app_status(app_id);
CREATE INDEX idx_app_status_timestamp ON app_status(timestamp);
CREATE INDEX idx_app_metrics_app_id ON app_metrics(app_id);
CREATE INDEX idx_app_metrics_timestamp ON app_metrics(timestamp);
CREATE INDEX idx_app_logs_app_id ON app_logs(app_id);
CREATE INDEX idx_app_logs_timestamp ON app_logs(timestamp);