-- System Monitor Database Schema

CREATE TABLE IF NOT EXISTS metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name VARCHAR(100) NOT NULL,
    value FLOAT NOT NULL,
    unit VARCHAR(50),
    source VARCHAR(100),
    tags JSONB,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT,
    metrics JSONB,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS investigations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    anomaly_id UUID REFERENCES anomalies(id) ON DELETE CASCADE,
    triggered_by VARCHAR(100),
    claude_request TEXT,
    claude_response TEXT,
    findings JSONB,
    recommendations JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    summary JSONB,
    metrics_snapshot JSONB,
    anomaly_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    period_start TIMESTAMP,
    period_end TIMESTAMP
);

CREATE TABLE IF NOT EXISTS thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name VARCHAR(100) NOT NULL UNIQUE,
    warning_threshold FLOAT,
    critical_threshold FLOAT,
    check_interval INTEGER DEFAULT 60,
    action VARCHAR(100),
    enabled BOOLEAN DEFAULT true,
    metadata JSONB
);

-- System health tracking
CREATE TABLE IF NOT EXISTS system_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cpu_usage FLOAT,
    memory_usage FLOAT,
    memory_total BIGINT,
    disk_usage FLOAT,
    disk_total BIGINT,
    network_rx_bytes BIGINT,
    network_tx_bytes BIGINT,
    tcp_connections INTEGER,
    process_count INTEGER,
    load_average FLOAT[],
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp);
CREATE INDEX idx_metrics_name ON metrics(metric_name);
CREATE INDEX idx_anomalies_status ON anomalies(status);
CREATE INDEX idx_anomalies_detected ON anomalies(detected_at);
CREATE INDEX idx_investigations_anomaly ON investigations(anomaly_id);
CREATE INDEX idx_system_health_timestamp ON system_health(timestamp);

-- Default thresholds
INSERT INTO thresholds (metric_name, warning_threshold, critical_threshold, action) VALUES
    ('cpu_usage', 70, 90, 'alert'),
    ('memory_usage', 80, 95, 'investigate'),
    ('disk_usage', 85, 95, 'alert'),
    ('tcp_connections', 500, 1000, 'investigate')
ON CONFLICT (metric_name) DO NOTHING;