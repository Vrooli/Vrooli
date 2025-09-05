-- Accessibility Compliance Hub Database Schema
-- Version 1.0.0

-- Create database if not exists
CREATE DATABASE IF NOT EXISTS accessibility_hub;

-- Use the database
\c accessibility_hub;

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Audit Reports Table
CREATE TABLE IF NOT EXISTS audit_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id VARCHAR(255) NOT NULL,
    scenario_name VARCHAR(255),
    wcag_level VARCHAR(3) CHECK (wcag_level IN ('A', 'AA', 'AAA')),
    score INTEGER CHECK (score >= 0 AND score <= 100),
    total_issues INTEGER DEFAULT 0,
    critical_issues INTEGER DEFAULT 0,
    major_issues INTEGER DEFAULT 0,
    minor_issues INTEGER DEFAULT 0,
    auto_fixable_issues INTEGER DEFAULT 0,
    fixes_applied INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed')),
    report_data JSONB,
    screenshot_path TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    INDEX idx_scenario_id (scenario_id),
    INDEX idx_created_at (created_at),
    INDEX idx_status (status)
);

-- Accessibility Issues Table
CREATE TABLE IF NOT EXISTS accessibility_issues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_id UUID REFERENCES audit_reports(id) ON DELETE CASCADE,
    issue_type VARCHAR(100) NOT NULL,
    wcag_criterion VARCHAR(20),
    severity VARCHAR(20) CHECK (severity IN ('critical', 'major', 'minor', 'warning')),
    impact_score INTEGER,
    element_selector TEXT,
    element_html TEXT,
    description TEXT,
    help_text TEXT,
    help_url TEXT,
    suggested_fix TEXT,
    ai_suggestion TEXT,
    auto_fixable BOOLEAN DEFAULT FALSE,
    fix_applied BOOLEAN DEFAULT FALSE,
    fix_verified BOOLEAN DEFAULT FALSE,
    occurrence_count INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_report_id (report_id),
    INDEX idx_issue_type (issue_type),
    INDEX idx_severity (severity),
    INDEX idx_auto_fixable (auto_fixable)
);

-- Remediation History Table
CREATE TABLE IF NOT EXISTS remediation_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issue_id UUID REFERENCES accessibility_issues(id) ON DELETE CASCADE,
    scenario_id VARCHAR(255) NOT NULL,
    fix_type VARCHAR(100),
    before_html TEXT,
    after_html TEXT,
    fix_method VARCHAR(50) CHECK (fix_method IN ('automatic', 'manual', 'ai_assisted')),
    success BOOLEAN,
    error_message TEXT,
    applied_by VARCHAR(255),
    verified_by VARCHAR(255),
    rollback_available BOOLEAN DEFAULT TRUE,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP WITH TIME ZONE,
    INDEX idx_issue_id (issue_id),
    INDEX idx_scenario_id (scenario_id),
    INDEX idx_applied_at (applied_at)
);

-- Accessible Patterns Table
CREATE TABLE IF NOT EXISTS accessible_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pattern_name VARCHAR(255) UNIQUE NOT NULL,
    pattern_type VARCHAR(100) NOT NULL,
    category VARCHAR(100),
    description TEXT,
    html_template TEXT,
    css_template TEXT,
    js_template TEXT,
    aria_attributes JSONB,
    wcag_criteria JSONB,
    usage_count INTEGER DEFAULT 0,
    success_rate NUMERIC(5,2),
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_pattern_type (pattern_type),
    INDEX idx_category (category),
    INDEX idx_usage_count (usage_count)
);

-- Compliance Metrics Table
CREATE TABLE IF NOT EXISTS compliance_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id VARCHAR(255) NOT NULL,
    metric_date DATE NOT NULL,
    average_score NUMERIC(5,2),
    total_audits INTEGER DEFAULT 0,
    total_issues_found INTEGER DEFAULT 0,
    total_issues_fixed INTEGER DEFAULT 0,
    auto_fix_rate NUMERIC(5,2),
    critical_issues_trend INTEGER[],
    score_trend NUMERIC[],
    most_common_issues JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scenario_id, metric_date),
    INDEX idx_scenario_metric (scenario_id, metric_date)
);

-- Pattern Usage Table (for ML training)
CREATE TABLE IF NOT EXISTS pattern_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pattern_id UUID REFERENCES accessible_patterns(id) ON DELETE CASCADE,
    scenario_id VARCHAR(255),
    context TEXT,
    effectiveness_score NUMERIC(3,2),
    user_feedback TEXT,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_pattern_id (pattern_id),
    INDEX idx_effectiveness (effectiveness_score)
);

-- Audit Schedule Table
CREATE TABLE IF NOT EXISTS audit_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id VARCHAR(255) NOT NULL,
    schedule_type VARCHAR(50) CHECK (schedule_type IN ('daily', 'weekly', 'monthly', 'on_deploy', 'manual')),
    cron_expression VARCHAR(100),
    last_run TIMESTAMP WITH TIME ZONE,
    next_run TIMESTAMP WITH TIME ZONE,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scenario_id),
    INDEX idx_next_run (next_run),
    INDEX idx_enabled (enabled)
);

-- Compliance Reports Table (for VPAT/documentation)
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id VARCHAR(255) NOT NULL,
    report_type VARCHAR(50) CHECK (report_type IN ('vpat', 'wcag', 'section508', 'ada', 'custom')),
    report_format VARCHAR(20) CHECK (report_format IN ('pdf', 'html', 'json', 'xml')),
    report_data JSONB,
    report_url TEXT,
    generated_by VARCHAR(255),
    valid_until DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_scenario_reports (scenario_id),
    INDEX idx_report_type (report_type),
    INDEX idx_created_at (created_at)
);

-- Create materialized view for dashboard
CREATE MATERIALIZED VIEW IF NOT EXISTS dashboard_overview AS
SELECT 
    s.scenario_id,
    s.scenario_name,
    COUNT(DISTINCT ar.id) as total_audits,
    AVG(ar.score)::NUMERIC(5,2) as average_score,
    MAX(ar.created_at) as last_audit_date,
    SUM(ar.total_issues) as total_issues_all_time,
    SUM(ar.fixes_applied) as total_fixes_applied,
    (SUM(ar.fixes_applied)::FLOAT / NULLIF(SUM(ar.auto_fixable_issues), 0) * 100)::NUMERIC(5,2) as auto_fix_success_rate
FROM 
    (SELECT DISTINCT scenario_id, scenario_name FROM audit_reports) s
LEFT JOIN 
    audit_reports ar ON s.scenario_id = ar.scenario_id
GROUP BY 
    s.scenario_id, s.scenario_name
WITH DATA;

-- Create index on materialized view
CREATE INDEX idx_dashboard_scenario ON dashboard_overview(scenario_id);

-- Refresh materialized view function
CREATE OR REPLACE FUNCTION refresh_dashboard_overview()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY dashboard_overview;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update trigger to relevant tables
CREATE TRIGGER update_audit_reports_updated_at 
    BEFORE UPDATE ON audit_reports 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_accessible_patterns_updated_at 
    BEFORE UPDATE ON accessible_patterns 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_audit_schedules_updated_at 
    BEFORE UPDATE ON audit_schedules 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default accessible patterns
INSERT INTO accessible_patterns (pattern_name, pattern_type, category, description, html_template, aria_attributes) VALUES
('accessible-button', 'component', 'interactive', 'Fully accessible button with proper ARIA', '<button type="button" aria-label="{{label}}" aria-pressed="{{pressed}}">{{text}}</button>', '{"aria-label": "required", "aria-pressed": "optional"}'),
('skip-to-content', 'navigation', 'landmark', 'Skip to main content link', '<a href="#main" class="skip-link">Skip to main content</a>', '{}'),
('accessible-form-field', 'component', 'form', 'Form field with proper labeling', '<div class="form-field"><label for="{{id}}">{{label}}</label><input type="{{type}}" id="{{id}}" name="{{name}}" aria-required="{{required}}" /></div>', '{"aria-required": "optional", "aria-invalid": "optional"}'),
('landmark-navigation', 'navigation', 'landmark', 'Main navigation with ARIA landmarks', '<nav role="navigation" aria-label="Main navigation"><ul>{{items}}</ul></nav>', '{"aria-label": "required"}')
ON CONFLICT (pattern_name) DO NOTHING;

-- Grant appropriate permissions
GRANT ALL PRIVILEGES ON DATABASE accessibility_hub TO postgres;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;