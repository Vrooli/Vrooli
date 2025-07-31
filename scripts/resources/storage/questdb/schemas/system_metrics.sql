-- System-wide performance and health metrics
-- Used for monitoring overall system performance

CREATE TABLE IF NOT EXISTS system_metrics (
    timestamp TIMESTAMP,                    -- Time of measurement
    host SYMBOL,                           -- Hostname or container name
    metric_name SYMBOL,                    -- Metric identifier (cpu_usage, memory_usage, etc.)
    metric_value DOUBLE,                   -- Numeric value of the metric
    metric_unit SYMBOL,                    -- Unit of measurement (percent, bytes, ms, etc.)
    component SYMBOL,                      -- System component (api, database, cache, etc.)
    tags STRING                           -- Additional tags as JSON
) timestamp(timestamp) PARTITION BY DAY;