package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func InitializeDatabase(db *sql.DB) error {
	schema := `
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
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_scanned TIMESTAMP WITH TIME ZONE,
    CONSTRAINT unique_target UNIQUE(address, port, protocol)
);

-- Scan results table
CREATE TABLE IF NOT EXISTS scan_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES network_targets(id) ON DELETE CASCADE,
    scan_type VARCHAR(50) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) DEFAULT 'running',
    results JSONB DEFAULT '{}',
    findings JSONB DEFAULT '{}',
    severity_score DECIMAL(3,1),
    recommendations JSONB DEFAULT '[]',
    raw_data_path TEXT,
    scan_config JSONB DEFAULT '{}'
);

-- DNS queries table
CREATE TABLE IF NOT EXISTS dns_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query VARCHAR(255) NOT NULL,
    record_type VARCHAR(20) NOT NULL,
    dns_server VARCHAR(100),
    response_time_ms INTEGER,
    answers JSONB DEFAULT '[]',
    authoritative BOOLEAN DEFAULT false,
    dnssec_valid BOOLEAN DEFAULT false,
    cached BOOLEAN DEFAULT false,
    query_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- HTTP requests table
CREATE TABLE IF NOT EXISTS http_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL,
    method VARCHAR(20) NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    headers JSONB DEFAULT '{}',
    response_headers JSONB DEFAULT '{}',
    response_body TEXT,
    error_message TEXT,
    request_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Port scan results
CREATE TABLE IF NOT EXISTS port_scan_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    protocol VARCHAR(10) NOT NULL,
    state VARCHAR(20) NOT NULL,
    service VARCHAR(100),
    version VARCHAR(100),
    banner TEXT,
    scan_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- SSL certificates
CREATE TABLE IF NOT EXISTS ssl_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname VARCHAR(255) NOT NULL,
    port INTEGER DEFAULT 443,
    subject TEXT,
    issuer TEXT,
    serial_number VARCHAR(100),
    not_before TIMESTAMP WITH TIME ZONE,
    not_after TIMESTAMP WITH TIME ZONE,
    signature_algorithm VARCHAR(50),
    key_usage VARCHAR(255),
    dns_names TEXT[],
    ip_addresses TEXT[],
    is_valid BOOLEAN,
    validation_errors TEXT[],
    check_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Monitoring metrics
CREATE TABLE IF NOT EXISTS monitoring_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES network_targets(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    value DECIMAL(15,6),
    unit VARCHAR(20),
    labels JSONB DEFAULT '{}',
    alert_level VARCHAR(20)
);

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES network_targets(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT
);

-- API definitions table for API testing
CREATE TABLE IF NOT EXISTS api_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    base_url TEXT NOT NULL,
    version VARCHAR(50),
    specification VARCHAR(50) CHECK (specification IN ('openapi', 'swagger', 'graphql', 'grpc')),
    spec_document JSONB DEFAULT '{}',
    authentication_methods TEXT[],
    rate_limits JSONB DEFAULT '{}',
    endpoints_count INTEGER DEFAULT 0,
    last_validated TIMESTAMP WITH TIME ZONE,
    validation_status VARCHAR(20) CHECK (validation_status IN ('valid', 'invalid', 'unknown')),
    documentation_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_network_targets_active ON network_targets(is_active);
CREATE INDEX IF NOT EXISTS idx_network_targets_type ON network_targets(target_type);
CREATE INDEX IF NOT EXISTS idx_scan_results_target ON scan_results(target_id);
CREATE INDEX IF NOT EXISTS idx_scan_results_type ON scan_results(scan_type);
CREATE INDEX IF NOT EXISTS idx_scan_results_status ON scan_results(status);
CREATE INDEX IF NOT EXISTS idx_dns_queries_query ON dns_queries(query);
CREATE INDEX IF NOT EXISTS idx_http_requests_url ON http_requests(url);
CREATE INDEX IF NOT EXISTS idx_http_requests_timestamp ON http_requests(request_timestamp);
CREATE INDEX IF NOT EXISTS idx_port_scan_target ON port_scan_results(target);
CREATE INDEX IF NOT EXISTS idx_ssl_certificates_hostname ON ssl_certificates(hostname);
CREATE INDEX IF NOT EXISTS idx_monitoring_metrics_target ON monitoring_metrics(target_id);
CREATE INDEX IF NOT EXISTS idx_monitoring_metrics_timestamp ON monitoring_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_alerts_target ON alerts(target_id);
CREATE INDEX IF NOT EXISTS idx_alerts_active ON alerts(is_active);
`

	// Execute the schema
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}