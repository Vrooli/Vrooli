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
    config JSONB,
    consecutive_failures INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS app_view_stats (
    scenario_name VARCHAR(255) PRIMARY KEY,
    view_count BIGINT DEFAULT 0,
    first_viewed_at TIMESTAMP,
    last_viewed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    cpu_usage FLOAT,
    memory_usage FLOAT,
    disk_usage FLOAT,
    network_in BIGINT,
    network_out BIGINT,
    response_time INTEGER,
    request_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
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

-- New tables for enhanced monitoring
CREATE TABLE IF NOT EXISTS app_health_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE UNIQUE,
    app_name VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    health_score INTEGER,
    status_code INTEGER,
    response_time INTEGER,
    consecutive_failures INTEGER DEFAULT 0,
    last_check TIMESTAMP,
    error_message TEXT,
    ai_analysis TEXT,
    ai_analyzed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS app_health_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    app_name VARCHAR(255),
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT,
    ai_analysis TEXT,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS app_performance_analysis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    app_name VARCHAR(255),
    performance_score INTEGER,
    avg_cpu FLOAT,
    max_cpu FLOAT,
    avg_memory FLOAT,
    max_memory FLOAT,
    avg_response_time FLOAT,
    max_response_time FLOAT,
    total_requests BIGINT,
    total_errors BIGINT,
    error_rate FLOAT,
    anomalies JSONB,
    anomaly_count INTEGER DEFAULT 0,
    ai_recommendations JSONB,
    time_range VARCHAR(10),
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_app_status_app_id ON app_status(app_id);
CREATE INDEX idx_app_status_timestamp ON app_status(timestamp);
CREATE INDEX idx_app_metrics_app_id ON app_metrics(app_id);
CREATE INDEX idx_app_metrics_timestamp ON app_metrics(timestamp);
CREATE INDEX idx_app_logs_app_id ON app_logs(app_id);
CREATE INDEX idx_app_logs_timestamp ON app_logs(timestamp);
CREATE INDEX idx_app_health_status_app_id ON app_health_status(app_id);
CREATE INDEX idx_app_health_alerts_app_id ON app_health_alerts(app_id);
CREATE INDEX idx_app_health_alerts_severity ON app_health_alerts(severity);
CREATE INDEX idx_app_performance_app_id ON app_performance_analysis(app_id);
CREATE INDEX idx_app_performance_timestamp ON app_performance_analysis(analyzed_at);
CREATE INDEX idx_app_view_stats_last_viewed ON app_view_stats(last_viewed_at DESC);
CREATE INDEX idx_app_view_stats_view_count ON app_view_stats(view_count DESC);
