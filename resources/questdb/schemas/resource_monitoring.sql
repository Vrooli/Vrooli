-- Resource health and availability monitoring
-- Tracks status and performance of all Vrooli resources

CREATE TABLE IF NOT EXISTS resource_health (
    timestamp TIMESTAMP,                    -- Time of health check
    resource_name SYMBOL,                  -- Resource identifier (ollama, n8n, questdb, etc.)
    resource_type SYMBOL,                  -- Category (ai, automation, storage, agent, search)
    status SYMBOL,                         -- Health status (healthy, degraded, unhealthy, offline)
    response_time_ms DOUBLE,               -- Health check response time
    uptime_seconds LONG,                   -- Time since last restart
    cpu_percent DOUBLE,                    -- CPU usage percentage
    memory_mb DOUBLE,                      -- Memory usage in MB
    memory_percent DOUBLE,                 -- Memory usage percentage
    disk_usage_mb DOUBLE,                  -- Disk space used in MB
    network_rx_bytes LONG,                 -- Network bytes received
    network_tx_bytes LONG,                 -- Network bytes transmitted
    active_connections INT,                -- Number of active connections
    error_count INT,                       -- Errors in check interval
    warning_count INT,                     -- Warnings in check interval
    port INT,                             -- Service port number
    version STRING,                        -- Resource version
    metadata STRING                        -- Additional metadata as JSON
) timestamp(timestamp) PARTITION BY DAY;