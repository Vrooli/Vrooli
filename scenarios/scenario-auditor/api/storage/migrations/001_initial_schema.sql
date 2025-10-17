-- Scenario Auditor Database Schema
-- Stores rule definitions, scan results, user preferences, and audit history

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Scenarios table - Master list of all scenarios to audit
CREATE TABLE scenarios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    path TEXT NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',
    version VARCHAR(20),
    tags TEXT[],
    last_audited TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Rule categories table - Organization of audit rules
CREATE TABLE rule_categories (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(20),
    icon VARCHAR(50),
    display_order INTEGER,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Rules table - Audit rule definitions
CREATE TABLE rules (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category_id VARCHAR(50) REFERENCES rule_categories(id) ON DELETE CASCADE,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    enabled BOOLEAN DEFAULT true,
    version VARCHAR(20) DEFAULT '1.0.0',
    condition_type VARCHAR(50) NOT NULL,
    condition_data JSONB,
    targets TEXT[],
    tags TEXT[],
    documentation TEXT,
    examples TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Scan results table - Results of audit scans
CREATE TABLE scan_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    scan_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'completed',
    rules_applied TEXT[],
    total_violations INTEGER DEFAULT 0,
    critical_violations INTEGER DEFAULT 0,
    high_violations INTEGER DEFAULT 0,
    medium_violations INTEGER DEFAULT 0,
    low_violations INTEGER DEFAULT 0,
    info_violations INTEGER DEFAULT 0,
    scan_duration_ms INTEGER,
    triggered_by VARCHAR(100),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Rule violations table - Individual rule violations found during scans  
CREATE TABLE rule_violations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_result_id UUID REFERENCES scan_results(id) ON DELETE CASCADE,
    rule_id VARCHAR(255) REFERENCES rules(id) ON DELETE CASCADE,
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    file_path TEXT,
    line_number INTEGER,
    column_number INTEGER,
    code_snippet TEXT,
    suggestion TEXT,
    status VARCHAR(50) DEFAULT 'open',
    fixed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User preferences table - Audit configuration preferences
CREATE TABLE user_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    preference_type VARCHAR(50) NOT NULL,
    enabled_rules TEXT[],
    disabled_categories TEXT[],
    scan_options JSONB,
    ui_settings JSONB,
    notification_settings JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- AI fixes table - AI-generated fixes for violations
CREATE TABLE ai_fixes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    violation_id UUID REFERENCES rule_violations(id) ON DELETE CASCADE,
    fix_type VARCHAR(50) NOT NULL,
    original_code TEXT,
    suggested_code TEXT,
    explanation TEXT,
    confidence_score DECIMAL(3,2),
    status VARCHAR(50) DEFAULT 'pending',
    applied_at TIMESTAMP,
    success BOOLEAN,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Audit metrics table - Performance and quality metrics
CREATE TABLE audit_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    value DECIMAL(10,3),
    unit VARCHAR(20),
    context JSONB,
    measured_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_scenarios_name ON scenarios(name);
CREATE INDEX idx_scenarios_status ON scenarios(status);
CREATE INDEX idx_rules_category ON rules(category_id);
CREATE INDEX idx_rules_enabled ON rules(enabled);
CREATE INDEX idx_rules_severity ON rules(severity);
CREATE INDEX idx_scan_results_scenario ON scan_results(scenario_id);
CREATE INDEX idx_scan_results_status ON scan_results(status);
CREATE INDEX idx_rule_violations_scan ON rule_violations(scan_result_id);
CREATE INDEX idx_rule_violations_rule ON rule_violations(rule_id);
CREATE INDEX idx_rule_violations_scenario ON rule_violations(scenario_id);
CREATE INDEX idx_rule_violations_severity ON rule_violations(severity);
CREATE INDEX idx_rule_violations_status ON rule_violations(status);
CREATE INDEX idx_ai_fixes_violation ON ai_fixes(violation_id);
CREATE INDEX idx_ai_fixes_status ON ai_fixes(status);
CREATE INDEX idx_audit_metrics_scenario ON audit_metrics(scenario_id);
CREATE INDEX idx_audit_metrics_type ON audit_metrics(metric_type);

-- Full-text search indexes
CREATE INDEX idx_rules_search ON rules USING gin(to_tsvector('english', name || ' ' || COALESCE(description, '')));
CREATE INDEX idx_rule_violations_search ON rule_violations USING gin(to_tsvector('english', message || ' ' || COALESCE(suggestion, '')));

-- Triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_scenarios_updated_at BEFORE UPDATE ON scenarios FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rules_updated_at BEFORE UPDATE ON rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON user_preferences FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert initial rule categories
INSERT INTO rule_categories (id, name, description, color, icon, display_order) VALUES
('api', 'API Standards', 'Go API best practices, security patterns, and documentation requirements', '#10B981', 'server', 1),
('config', 'Configuration Standards', 'service.json schema compliance and lifecycle completeness', '#F59E0B', 'settings', 2),
('ui', 'UI Standards', 'Browserless testing practices, accessibility, and performance', '#3B82F6', 'monitor', 3),
('testing', 'Testing Standards', 'Phase-based structure, coverage requirements, and integration patterns', '#8B5CF6', 'check-circle', 4)
ON CONFLICT (id) DO NOTHING;

-- Insert initial rules based on existing rule files
INSERT INTO rules (id, name, description, category_id, severity, condition_type, condition_data, targets, tags) VALUES
('config-service-json-exists', 'Service JSON File Exists', 'Ensures that .vrooli/service.json file exists in the scenario', 'config', 'critical', 'file_exists', '{"path": ".vrooli/service.json"}', ARRAY['scenarios'], ARRAY['lifecycle', 'required']),
('ui-unique-element-ids', 'Unique Element IDs', 'Ensures UI elements have unique IDs for reliable testing', 'ui', 'medium', 'pattern_match', '{"pattern": "id=[\"\'][^\"\']+[\"\']", "scope": "ui"}', ARRAY['ui', 'html', 'jsx'], ARRAY['testing', 'accessibility']),
('api-error-handling', 'Proper Error Handling', 'API endpoints should handle errors gracefully', 'api', 'high', 'pattern_match', '{"pattern": "if err != nil", "scope": "api"}', ARRAY['go', 'api'], ARRAY['error-handling', 'reliability']),
('testing-unit-coverage', 'Unit Test Coverage', 'Ensures adequate unit test coverage', 'testing', 'medium', 'coverage_check', '{"minimum_coverage": 70}', ARRAY['tests'], ARRAY['quality', 'coverage'])
ON CONFLICT (id) DO NOTHING;

-- Views for common queries
CREATE VIEW v_scenario_audit_health AS
SELECT 
    s.name,
    s.status,
    s.last_audited,
    COALESCE(rv.total_violations, 0) as total_violations,
    COALESCE(rv.critical_violations, 0) as critical_violations,
    COALESCE(rv.high_violations, 0) as high_violations,
    COALESCE(rv.medium_violations, 0) as medium_violations,
    COALESCE(rv.low_violations, 0) as low_violations,
    CASE 
        WHEN rv.critical_violations > 0 THEN 'critical'
        WHEN rv.high_violations > 5 THEN 'degraded'
        WHEN rv.total_violations = 0 THEN 'healthy'
        ELSE 'acceptable'
    END as health_status
FROM scenarios s
LEFT JOIN (
    SELECT 
        scenario_id,
        COUNT(*) as total_violations,
        COUNT(CASE WHEN severity = 'critical' THEN 1 END) as critical_violations,
        COUNT(CASE WHEN severity = 'high' THEN 1 END) as high_violations,
        COUNT(CASE WHEN severity = 'medium' THEN 1 END) as medium_violations,
        COUNT(CASE WHEN severity = 'low' THEN 1 END) as low_violations
    FROM rule_violations 
    WHERE status = 'open'
    GROUP BY scenario_id
) rv ON s.id = rv.scenario_id;

CREATE VIEW v_rule_effectiveness AS
SELECT 
    r.id,
    r.name,
    r.category_id,
    r.severity,
    COUNT(rv.id) as violations_found,
    COUNT(CASE WHEN rv.status = 'fixed' THEN 1 END) as violations_fixed,
    CASE 
        WHEN COUNT(rv.id) = 0 THEN 0
        ELSE (COUNT(CASE WHEN rv.status = 'fixed' THEN 1 END)::decimal / COUNT(rv.id) * 100)
    END as fix_rate_percent
FROM rules r
LEFT JOIN rule_violations rv ON r.id = rv.rule_id
WHERE r.enabled = true
GROUP BY r.id, r.name, r.category_id, r.severity
ORDER BY violations_found DESC;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;