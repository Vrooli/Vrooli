-- QuestDB Time-Series Tables for Resource Monitoring
-- High-performance storage for metrics and events

-- ============================================
-- Resource Metrics Table
-- ============================================
-- Stores all resource metrics with high-frequency sampling
CREATE TABLE IF NOT EXISTS resource_metrics (
    timestamp TIMESTAMP,
    resource_name SYMBOL,
    resource_type SYMBOL,
    metric_type SYMBOL,
    value DOUBLE,
    status SYMBOL,
    response_time_ms INT,
    error_count INT,
    tags STRING
) timestamp(timestamp) PARTITION BY DAY WAL;

-- Create indexes for efficient querying
-- QuestDB automatically indexes the designated timestamp column
-- SYMBOL columns are automatically indexed for fast lookups

-- ============================================
-- Alert Events Table
-- ============================================
-- Stores all alert events for analysis and correlation
CREATE TABLE IF NOT EXISTS alert_events (
    timestamp TIMESTAMP,
    resource_name SYMBOL,
    alert_type SYMBOL,
    severity SYMBOL,
    message STRING,
    metric_value DOUBLE,
    threshold_value DOUBLE,
    notified BOOLEAN,
    notification_channels STRING,
    alert_rule_id INT,
    context STRING
) timestamp(timestamp) PARTITION BY DAY WAL;

-- ============================================
-- System Events Table
-- ============================================
-- Tracks system-level events like discovery, configuration changes
CREATE TABLE IF NOT EXISTS system_events (
    timestamp TIMESTAMP,
    event_type SYMBOL,
    event_source SYMBOL,
    event_action SYMBOL,
    resource_name SYMBOL,
    user_id SYMBOL,
    success BOOLEAN,
    duration_ms INT,
    details STRING,
    error_message STRING
) timestamp(timestamp) PARTITION BY DAY WAL;

-- ============================================
-- Performance Metrics Table
-- ============================================
-- Detailed performance metrics for each resource
CREATE TABLE IF NOT EXISTS performance_metrics (
    timestamp TIMESTAMP,
    resource_name SYMBOL,
    cpu_usage_percent DOUBLE,
    memory_usage_mb DOUBLE,
    disk_io_read_mb DOUBLE,
    disk_io_write_mb DOUBLE,
    network_in_mb DOUBLE,
    network_out_mb DOUBLE,
    open_connections INT,
    active_requests INT,
    queue_size INT,
    error_rate DOUBLE
) timestamp(timestamp) PARTITION BY HOUR WAL;

-- ============================================
-- API Metrics Table
-- ============================================
-- Tracks API endpoint performance for each resource
CREATE TABLE IF NOT EXISTS api_metrics (
    timestamp TIMESTAMP,
    resource_name SYMBOL,
    endpoint SYMBOL,
    method SYMBOL,
    status_code INT,
    response_time_ms INT,
    request_size_bytes LONG,
    response_size_bytes LONG,
    user_agent STRING,
    error_type SYMBOL
) timestamp(timestamp) PARTITION BY DAY WAL;

-- ============================================
-- Aggregated Metrics Tables (for faster queries)
-- ============================================

-- 5-minute aggregations
CREATE TABLE IF NOT EXISTS metrics_5min (
    timestamp TIMESTAMP,
    resource_name SYMBOL,
    resource_type SYMBOL,
    availability_percent DOUBLE,
    avg_response_time_ms DOUBLE,
    max_response_time_ms DOUBLE,
    min_response_time_ms DOUBLE,
    error_count LONG,
    request_count LONG,
    p50_response_time_ms DOUBLE,
    p95_response_time_ms DOUBLE,
    p99_response_time_ms DOUBLE
) timestamp(timestamp) PARTITION BY DAY WAL;

-- Hourly aggregations
CREATE TABLE IF NOT EXISTS metrics_hourly (
    timestamp TIMESTAMP,
    resource_name SYMBOL,
    resource_type SYMBOL,
    availability_percent DOUBLE,
    avg_response_time_ms DOUBLE,
    max_response_time_ms DOUBLE,
    min_response_time_ms DOUBLE,
    total_errors LONG,
    total_requests LONG,
    p50_response_time_ms DOUBLE,
    p95_response_time_ms DOUBLE,
    p99_response_time_ms DOUBLE,
    alert_count INT
) timestamp(timestamp) PARTITION BY MONTH WAL;

-- Daily aggregations
CREATE TABLE IF NOT EXISTS metrics_daily (
    timestamp TIMESTAMP,
    resource_name SYMBOL,
    resource_type SYMBOL,
    availability_percent DOUBLE,
    avg_response_time_ms DOUBLE,
    max_response_time_ms DOUBLE,
    min_response_time_ms DOUBLE,
    total_errors LONG,
    total_requests LONG,
    uptime_minutes INT,
    downtime_minutes INT,
    alert_count INT,
    critical_alert_count INT
) timestamp(timestamp) PARTITION BY YEAR WAL;

-- ============================================
-- Monitoring Dashboard Queries Table
-- ============================================
-- Stores custom queries for dashboard widgets
CREATE TABLE IF NOT EXISTS dashboard_queries (
    timestamp TIMESTAMP,
    query_id SYMBOL,
    query_name STRING,
    query_sql STRING,
    widget_type SYMBOL,
    refresh_interval_seconds INT,
    created_by SYMBOL,
    is_active BOOLEAN
) timestamp(timestamp) PARTITION BY MONTH WAL;

-- ============================================
-- Sample Data for Testing (Optional)
-- ============================================
-- Insert some sample data to verify tables are working

-- Sample resource metrics
INSERT INTO resource_metrics 
SELECT 
    dateadd('m', -x*5, now()) as timestamp,
    'postgres' as resource_name,
    'storage' as resource_type,
    'availability' as metric_type,
    1.0 as value,
    'healthy' as status,
    rnd_int(10, 50, 0) as response_time_ms,
    0 as error_count,
    'sample=true' as tags
FROM long_sequence(12);

INSERT INTO resource_metrics 
SELECT 
    dateadd('m', -x*5, now()) as timestamp,
    'n8n' as resource_name,
    'automation' as resource_type,
    'availability' as metric_type,
    1.0 as value,
    'healthy' as status,
    rnd_int(20, 100, 0) as response_time_ms,
    0 as error_count,
    'sample=true' as tags
FROM long_sequence(12);

-- Sample alert event
INSERT INTO alert_events VALUES(
    now(),
    'redis',
    'high_response_time',
    'warning',
    'Response time exceeded threshold',
    150.5,
    100.0,
    false,
    '["email"]',
    1,
    '{"test": true}'
);

-- Sample system event
INSERT INTO system_events VALUES(
    now(),
    'discovery',
    'auto_discovery',
    'resource_found',
    'windmill',
    'system',
    true,
    250,
    'Discovered new resource on port 5681',
    null
);

-- ============================================
-- Useful Queries for Monitoring
-- ============================================

-- These are example queries that can be used by the dashboard

-- Get current status of all resources (last value per resource)
-- SELECT resource_name, resource_type, status, value, response_time_ms, timestamp
-- FROM resource_metrics
-- WHERE timestamp > dateadd('m', -5, now())
-- LATEST ON timestamp PARTITION BY resource_name;

-- Get 5-minute average response times
-- SELECT timestamp, resource_name, 
--        avg(response_time_ms) as avg_response_time,
--        max(response_time_ms) as max_response_time,
--        count(*) as sample_count
-- FROM resource_metrics
-- WHERE timestamp > dateadd('h', -1, now())
-- SAMPLE BY 5m ALIGN TO CALENDAR;

-- Count alerts by severity in last 24 hours
-- SELECT severity, count(*) as alert_count
-- FROM alert_events
-- WHERE timestamp > dateadd('d', -1, now())
-- GROUP BY severity;

-- Get availability percentage per resource over last hour
-- SELECT resource_name,
--        100.0 * sum(CASE WHEN status = 'healthy' THEN 1 ELSE 0 END) / count(*) as availability_pct
-- FROM resource_metrics
-- WHERE timestamp > dateadd('h', -1, now())
-- GROUP BY resource_name;