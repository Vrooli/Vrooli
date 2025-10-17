-- Network Tools Database Schema
-- Core database structure for network operations and testing

-- Network targets table: Hosts, URLs, APIs to monitor/test
CREATE TABLE IF NOT EXISTS network_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    target_type VARCHAR(50) NOT NULL CHECK (target_type IN ('host', 'url', 'api', 'service')),
    address VARCHAR(500) NOT NULL,
    port INTEGER,
    protocol VARCHAR(20) CHECK (protocol IN ('http', 'https', 'tcp', 'udp', 'icmp')),
    authentication JSONB DEFAULT '{}',
    tags TEXT[],
    is_active BOOLEAN DEFAULT true,
    scan_schedule JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_scanned TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT unique_target UNIQUE(address, port, protocol)
);

-- Scan results table: Store all network scan results
CREATE TABLE IF NOT EXISTS scan_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES network_targets(id) ON DELETE CASCADE,
    scan_type VARCHAR(50) NOT NULL CHECK (scan_type IN ('port_scan', 'vulnerability_scan', 'ssl_check', 'dns_lookup', 'api_test', 'connectivity')),
    
    -- Scan details
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed', 'timeout', 'error')),
    
    -- Results
    results JSONB DEFAULT '{}',
    findings JSONB DEFAULT '{}',
    severity_score DECIMAL(3,1),
    recommendations JSONB DEFAULT '[]',
    raw_data_path TEXT,
    scan_config JSONB DEFAULT '{}',
    
    -- Performance metrics
    response_time_ms INTEGER,
    packet_loss_percent DECIMAL(5,2),
    
    -- Metadata
    scanner_version VARCHAR(50),
    triggered_by VARCHAR(255)
);

-- API definitions table: Store API specifications for testing
CREATE TABLE IF NOT EXISTS api_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    version VARCHAR(50),
    specification VARCHAR(50) CHECK (specification IN ('openapi', 'swagger', 'graphql', 'grpc', 'rest')),
    spec_document JSONB,
    authentication_methods TEXT[],
    rate_limits JSONB DEFAULT '{}',
    endpoints_count INTEGER,
    
    -- Validation
    last_validated TIMESTAMP WITH TIME ZONE,
    validation_status VARCHAR(50) CHECK (validation_status IN ('valid', 'invalid', 'unknown')),
    documentation_url TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT unique_api_version UNIQUE(base_url, version)
);

-- Monitoring data table: Time-series network metrics
CREATE TABLE IF NOT EXISTS monitoring_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES network_targets(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL CHECK (metric_type IN ('latency', 'throughput', 'availability', 'error_rate', 'bandwidth')),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    value DECIMAL(15,6) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    labels JSONB DEFAULT '{}',
    alert_level VARCHAR(20) CHECK (alert_level IN ('ok', 'warning', 'critical')),
    measurement_location VARCHAR(255),
    
    -- Index for time-series queries
    INDEX idx_monitoring_timestamp (target_id, timestamp DESC),
    INDEX idx_monitoring_metric (metric_type, timestamp DESC)
);

-- HTTP requests table: Log HTTP operations
CREATE TABLE IF NOT EXISTS http_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL,
    method VARCHAR(10) NOT NULL CHECK (method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS')),
    headers JSONB DEFAULT '{}',
    body TEXT,
    
    -- Response
    status_code INTEGER,
    response_headers JSONB DEFAULT '{}',
    response_body TEXT,
    response_time_ms INTEGER,
    final_url TEXT,
    
    -- SSL/TLS info
    ssl_info JSONB DEFAULT '{}',
    redirect_chain JSONB DEFAULT '[]',
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    user_agent TEXT,
    source_ip VARCHAR(50),
    
    -- Performance tracking
    INDEX idx_http_requests_timestamp (created_at DESC)
);

-- DNS queries table: Store DNS lookup results
CREATE TABLE IF NOT EXISTS dns_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query VARCHAR(500) NOT NULL,
    record_type VARCHAR(10) NOT NULL,
    dns_server VARCHAR(50),
    
    -- Results
    answers JSONB DEFAULT '[]',
    response_time_ms INTEGER,
    authoritative BOOLEAN,
    dnssec_valid BOOLEAN,
    
    -- Cache control
    ttl INTEGER,
    cached_until TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Cache lookup index
    INDEX idx_dns_cache (query, record_type, cached_until)
);

-- Port scans table: Detailed port scanning results
CREATE TABLE IF NOT EXISTS port_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES network_targets(id) ON DELETE CASCADE,
    scan_result_id UUID REFERENCES scan_results(id) ON DELETE CASCADE,
    
    port INTEGER NOT NULL,
    protocol VARCHAR(10) NOT NULL CHECK (protocol IN ('tcp', 'udp')),
    state VARCHAR(20) NOT NULL CHECK (state IN ('open', 'closed', 'filtered', 'unfiltered')),
    service VARCHAR(100),
    version VARCHAR(100),
    banner TEXT,
    
    -- Timing
    response_time_ms INTEGER,
    scanned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Index for port lookups
    INDEX idx_port_scans_target (target_id, port, protocol)
);

-- API test results table: Store API endpoint test results
CREATE TABLE IF NOT EXISTS api_test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_definition_id UUID REFERENCES api_definitions(id) ON DELETE CASCADE,
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL,
    
    -- Test execution
    test_name VARCHAR(255),
    test_suite VARCHAR(255),
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Results
    status VARCHAR(20) CHECK (status IN ('passed', 'failed', 'skipped', 'error')),
    expected_status INTEGER,
    actual_status INTEGER,
    response_time_ms INTEGER,
    
    -- Validation
    schema_valid BOOLEAN,
    assertions_passed INTEGER,
    assertions_failed INTEGER,
    error_details JSONB DEFAULT '{}',
    
    -- Performance tracking
    INDEX idx_api_tests_timestamp (api_definition_id, executed_at DESC)
);

-- SSL certificates table: Store SSL/TLS certificate information
CREATE TABLE IF NOT EXISTS ssl_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES network_targets(id) ON DELETE CASCADE,
    
    -- Certificate details
    common_name VARCHAR(255),
    issuer VARCHAR(500),
    subject VARCHAR(500),
    serial_number VARCHAR(100),
    
    -- Validity
    not_before TIMESTAMP WITH TIME ZONE,
    not_after TIMESTAMP WITH TIME ZONE,
    is_valid BOOLEAN,
    is_expired BOOLEAN,
    days_until_expiry INTEGER,
    
    -- Security
    signature_algorithm VARCHAR(100),
    key_size INTEGER,
    san_list JSONB DEFAULT '[]',
    chain_valid BOOLEAN,
    vulnerabilities JSONB DEFAULT '[]',
    
    -- Metadata
    scanned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    raw_certificate TEXT,
    
    -- Index for expiry tracking
    INDEX idx_ssl_expiry (not_after, is_valid)
);

-- Alerts table: Network monitoring alerts
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES network_targets(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    
    -- Alert details
    title VARCHAR(500) NOT NULL,
    message TEXT,
    threshold_value DECIMAL(15,6),
    actual_value DECIMAL(15,6),
    
    -- State
    is_active BOOLEAN DEFAULT true,
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_by VARCHAR(255),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Index for active alerts
    INDEX idx_active_alerts (is_active, severity, created_at DESC)
);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers
CREATE TRIGGER update_network_targets_updated_at BEFORE UPDATE
    ON network_targets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_api_definitions_updated_at BEFORE UPDATE
    ON api_definitions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alerts_updated_at BEFORE UPDATE
    ON alerts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create performance indexes
CREATE INDEX idx_scan_results_status ON scan_results(status, completed_at DESC);
CREATE INDEX idx_http_requests_url ON http_requests(url, created_at DESC);
CREATE INDEX idx_network_targets_active ON network_targets(is_active, last_scanned);

-- Create views for common queries
CREATE OR REPLACE VIEW active_targets AS
SELECT 
    nt.*,
    COUNT(sr.id) as total_scans,
    MAX(sr.completed_at) as last_scan_completed,
    AVG(sr.response_time_ms) as avg_response_time_ms
FROM network_targets nt
LEFT JOIN scan_results sr ON nt.id = sr.target_id AND sr.status = 'success'
WHERE nt.is_active = true
GROUP BY nt.id;

CREATE OR REPLACE VIEW recent_alerts AS
SELECT 
    a.*,
    nt.name as target_name,
    nt.address as target_address
FROM alerts a
JOIN network_targets nt ON a.target_id = nt.id
WHERE a.is_active = true
ORDER BY a.severity DESC, a.created_at DESC;

-- Add comments for documentation
COMMENT ON TABLE network_targets IS 'Network endpoints to monitor and test';
COMMENT ON TABLE scan_results IS 'Results from all types of network scans';
COMMENT ON TABLE api_definitions IS 'API specifications for testing';
COMMENT ON TABLE monitoring_data IS 'Time-series metrics for network performance';
COMMENT ON TABLE http_requests IS 'Log of HTTP/HTTPS requests made';
COMMENT ON TABLE dns_queries IS 'DNS lookup results with caching';
COMMENT ON TABLE port_scans IS 'Detailed port scanning results';
COMMENT ON TABLE ssl_certificates IS 'SSL/TLS certificate information and validation';
COMMENT ON TABLE alerts IS 'Network monitoring alerts and incidents';