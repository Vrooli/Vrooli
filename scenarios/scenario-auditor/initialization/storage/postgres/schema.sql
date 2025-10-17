-- API Manager Database Schema
-- Stores API metadata, scan results, vulnerabilities, and improvement tracking

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Scenarios table - Master list of all scenarios with APIs
CREATE TABLE scenarios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    path TEXT NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',
    api_port INTEGER,
    api_version VARCHAR(20),
    last_scanned TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API endpoints table - Individual endpoints discovered in each scenario
CREATE TABLE api_endpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    handler_function TEXT,
    line_number INTEGER,
    file_path TEXT,
    description TEXT,
    parameters JSONB,
    responses JSONB,
    security_requirements JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scenario_id, method, path)
);

-- Vulnerability scans table - Security audit results
CREATE TABLE vulnerability_scans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    scan_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    category VARCHAR(100) NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    file_path TEXT,
    line_number INTEGER,
    code_snippet TEXT,
    recommendation TEXT,
    status VARCHAR(50) DEFAULT 'open',
    fixed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Performance metrics table - API performance baselines and monitoring
CREATE TABLE performance_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    endpoint_id UUID REFERENCES api_endpoints(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    value DECIMAL(10,3),
    unit VARCHAR(20),
    measured_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    conditions JSONB
);

-- API documentation table - Generated OpenAPI specs and docs
CREATE TABLE api_documentation (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    doc_type VARCHAR(50) NOT NULL,
    title TEXT,
    content JSONB,
    version VARCHAR(20),
    auto_generated BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Improvement recommendations table - Suggested fixes and enhancements
CREATE TABLE improvements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    priority VARCHAR(20) NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    recommended_action TEXT,
    automated_fix JSONB,
    status VARCHAR(50) DEFAULT 'pending',
    applied_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Applied fixes table - Track what fixes have been applied
CREATE TABLE applied_fixes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    improvement_id UUID REFERENCES improvements(id) ON DELETE CASCADE,
    fix_type VARCHAR(50) NOT NULL,
    original_code TEXT,
    fixed_code TEXT,
    file_path TEXT,
    line_range TEXT,
    rollback_info JSONB,
    success BOOLEAN,
    error_message TEXT,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Scan history table - Track all scanning activities
CREATE TABLE scan_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    scan_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    results_summary JSONB,
    duration_ms INTEGER,
    error_message TEXT,
    triggered_by VARCHAR(100),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- API patterns table - Learned patterns for similarity analysis
CREATE TABLE api_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pattern_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pattern_data JSONB,
    examples TEXT[],
    quality_score DECIMAL(3,2),
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_scenarios_name ON scenarios(name);
CREATE INDEX idx_scenarios_status ON scenarios(status);
CREATE INDEX idx_api_endpoints_scenario ON api_endpoints(scenario_id);
CREATE INDEX idx_api_endpoints_method_path ON api_endpoints(method, path);
CREATE INDEX idx_vulnerability_scans_scenario ON vulnerability_scans(scenario_id);
CREATE INDEX idx_vulnerability_scans_severity ON vulnerability_scans(severity);
CREATE INDEX idx_vulnerability_scans_status ON vulnerability_scans(status);
CREATE INDEX idx_performance_metrics_scenario ON performance_metrics(scenario_id);
CREATE INDEX idx_performance_metrics_endpoint ON performance_metrics(endpoint_id);
CREATE INDEX idx_improvements_scenario ON improvements(scenario_id);
CREATE INDEX idx_improvements_status ON improvements(status);
CREATE INDEX idx_scan_history_scenario ON scan_history(scenario_id);
CREATE INDEX idx_scan_history_type ON scan_history(scan_type);

-- Full-text search indexes
CREATE INDEX idx_vulnerability_scans_search ON vulnerability_scans USING gin(to_tsvector('english', title || ' ' || description));
CREATE INDEX idx_improvements_search ON improvements USING gin(to_tsvector('english', title || ' ' || description));

-- Triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_scenarios_updated_at BEFORE UPDATE ON scenarios FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_api_documentation_updated_at BEFORE UPDATE ON api_documentation FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert initial data
INSERT INTO scenarios (name, path, description, status) 
SELECT 
    'system-bootstrap' as name,
    'internal' as path,
    'Initial system setup scenario' as description,
    'active' as status
WHERE NOT EXISTS (SELECT 1 FROM scenarios WHERE name = 'system-bootstrap');

-- Views for common queries
CREATE VIEW v_scenario_health AS
SELECT 
    s.name,
    s.status,
    s.last_scanned,
    COUNT(DISTINCT ae.id) as endpoint_count,
    COUNT(CASE WHEN vs.severity = 'high' THEN 1 END) as high_vulnerabilities,
    COUNT(CASE WHEN vs.severity = 'medium' THEN 1 END) as medium_vulnerabilities,
    COUNT(CASE WHEN vs.severity = 'low' THEN 1 END) as low_vulnerabilities,
    COUNT(CASE WHEN i.status = 'pending' THEN 1 END) as pending_improvements
FROM scenarios s
LEFT JOIN api_endpoints ae ON s.id = ae.scenario_id
LEFT JOIN vulnerability_scans vs ON s.id = vs.scenario_id AND vs.status = 'open'
LEFT JOIN improvements i ON s.id = i.scenario_id
GROUP BY s.id, s.name, s.status, s.last_scanned;

CREATE VIEW v_vulnerability_summary AS
SELECT 
    vs.severity,
    vs.category,
    COUNT(*) as count,
    COUNT(CASE WHEN vs.status = 'open' THEN 1 END) as open_count,
    COUNT(CASE WHEN vs.status = 'fixed' THEN 1 END) as fixed_count
FROM vulnerability_scans vs
GROUP BY vs.severity, vs.category
ORDER BY 
    CASE vs.severity 
        WHEN 'high' THEN 1 
        WHEN 'medium' THEN 2 
        WHEN 'low' THEN 3 
    END,
    vs.category;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;