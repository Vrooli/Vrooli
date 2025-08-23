-- App Debugger Database Schema

-- Error logs table
CREATE TABLE IF NOT EXISTS error_logs (
    id SERIAL PRIMARY KEY,
    app_name VARCHAR(255) NOT NULL,
    error_type VARCHAR(100),
    error_message TEXT,
    stack_trace TEXT,
    analysis JSONB,
    severity VARCHAR(20) DEFAULT 'unknown',
    resolved BOOLEAN DEFAULT FALSE,
    resolution_attempts INT DEFAULT 0,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Fix suggestions table
CREATE TABLE IF NOT EXISTS fix_suggestions (
    id SERIAL PRIMARY KEY,
    error_id INT REFERENCES error_logs(id) ON DELETE CASCADE,
    suggestions JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    applied BOOLEAN DEFAULT FALSE,
    success_rate DECIMAL(5,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Performance metrics table
CREATE TABLE IF NOT EXISTS performance_metrics (
    id SERIAL PRIMARY KEY,
    app_name VARCHAR(255) NOT NULL,
    cpu_usage DECIMAL(5,2),
    memory_usage DECIMAL(5,2),
    memory_limit VARCHAR(50),
    disk_usage DECIMAL(5,2),
    network_io JSONB,
    performance_status VARCHAR(20),
    recommendations TEXT[],
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Debug sessions table
CREATE TABLE IF NOT EXISTS debug_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(100) UNIQUE NOT NULL,
    app_name VARCHAR(255) NOT NULL,
    debug_type VARCHAR(50),
    status VARCHAR(50) DEFAULT 'active',
    report JSONB,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- App health history
CREATE TABLE IF NOT EXISTS app_health_history (
    id SERIAL PRIMARY KEY,
    app_name VARCHAR(255) NOT NULL,
    health_status VARCHAR(20) NOT NULL,
    error_count INT DEFAULT 0,
    warning_count INT DEFAULT 0,
    cpu_avg DECIMAL(5,2),
    memory_avg DECIMAL(5,2),
    uptime_seconds BIGINT,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_error_logs_app_name ON error_logs(app_name);
CREATE INDEX idx_error_logs_timestamp ON error_logs(timestamp);
CREATE INDEX idx_error_logs_severity ON error_logs(severity);
CREATE INDEX idx_performance_metrics_app_name ON performance_metrics(app_name);
CREATE INDEX idx_performance_metrics_timestamp ON performance_metrics(timestamp);
CREATE INDEX idx_debug_sessions_app_name ON debug_sessions(app_name);
CREATE INDEX idx_app_health_history_app_name ON app_health_history(app_name);

-- Create a view for recent errors
CREATE OR REPLACE VIEW recent_errors AS
SELECT 
    el.id,
    el.app_name,
    el.error_type,
    el.error_message,
    el.severity,
    el.resolved,
    el.timestamp,
    fs.suggestions,
    fs.status as fix_status
FROM error_logs el
LEFT JOIN fix_suggestions fs ON el.id = fs.error_id
WHERE el.timestamp > NOW() - INTERVAL '24 hours'
ORDER BY el.timestamp DESC;

-- Create a view for app health summary
CREATE OR REPLACE VIEW app_health_summary AS
SELECT 
    app_name,
    COUNT(CASE WHEN severity = 'critical' THEN 1 END) as critical_errors,
    COUNT(CASE WHEN severity = 'high' THEN 1 END) as high_errors,
    COUNT(CASE WHEN severity = 'medium' THEN 1 END) as medium_errors,
    COUNT(CASE WHEN severity = 'low' THEN 1 END) as low_errors,
    COUNT(CASE WHEN resolved = FALSE THEN 1 END) as unresolved_errors,
    MAX(timestamp) as last_error_time
FROM error_logs
WHERE timestamp > NOW() - INTERVAL '7 days'
GROUP BY app_name;