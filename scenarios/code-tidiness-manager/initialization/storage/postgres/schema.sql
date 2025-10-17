-- Code Tidiness Manager Database Schema
-- Stores scan history, cleanup patterns, and learning data

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS tidiness;

-- Set search path
SET search_path TO tidiness, public;

-- Scan history table
CREATE TABLE IF NOT EXISTS scan_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    scan_type VARCHAR(50) NOT NULL CHECK (scan_type IN ('quick', 'standard', 'deep', 'targeted')),
    paths TEXT[],
    issue_types TEXT[],
    total_files_scanned INTEGER DEFAULT 0,
    total_issues_found INTEGER DEFAULT 0,
    duration_ms INTEGER,
    status VARCHAR(20) DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::JSONB,
    created_by VARCHAR(100) DEFAULT 'system'
);

-- Scan results table
CREATE TABLE IF NOT EXISTS scan_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID NOT NULL REFERENCES scan_history(scan_id) ON DELETE CASCADE,
    issue_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    category VARCHAR(50) NOT NULL,
    file_path TEXT NOT NULL,
    line_number INTEGER,
    description TEXT NOT NULL,
    cleanup_script TEXT,
    confidence_score DECIMAL(3,2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    requires_human_review BOOLEAN DEFAULT false,
    estimated_impact VARCHAR(20) CHECK (estimated_impact IN ('trivial', 'minor', 'moderate', 'major')),
    safe_to_automate BOOLEAN DEFAULT false,
    files_affected TEXT[],
    size_bytes BIGINT,
    metadata JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scan_id, file_path, issue_type, line_number)
);

-- Cleanup actions table
CREATE TABLE IF NOT EXISTS cleanup_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_result_id UUID NOT NULL REFERENCES scan_results(id) ON DELETE CASCADE,
    action_taken VARCHAR(20) NOT NULL CHECK (action_taken IN ('accepted', 'rejected', 'modified', 'deferred', 'automated')),
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    executed_by VARCHAR(100) DEFAULT 'system',
    execution_result JSONB DEFAULT '{}'::JSONB,
    rollback_script TEXT,
    files_modified TEXT[],
    files_deleted TEXT[],
    bytes_freed BIGINT,
    user_feedback TEXT,
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Cleanup rules table
CREATE TABLE IF NOT EXISTS cleanup_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name VARCHAR(100) UNIQUE NOT NULL,
    category VARCHAR(50) NOT NULL,
    pattern TEXT NOT NULL,
    pattern_type VARCHAR(20) DEFAULT 'glob' CHECK (pattern_type IN ('glob', 'regex', 'exact')),
    action_template TEXT,
    description TEXT,
    priority INTEGER DEFAULT 50 CHECK (priority >= 0 AND priority <= 100),
    enabled BOOLEAN DEFAULT true,
    auto_apply BOOLEAN DEFAULT false,
    confidence_threshold DECIMAL(3,2) DEFAULT 0.90,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Learning feedback table
CREATE TABLE IF NOT EXISTS learning_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_result_id UUID REFERENCES scan_results(id) ON DELETE CASCADE,
    cleanup_rule_id UUID REFERENCES cleanup_rules(id) ON DELETE CASCADE,
    feedback_type VARCHAR(20) NOT NULL CHECK (feedback_type IN ('positive', 'negative', 'neutral')),
    feedback_score INTEGER CHECK (feedback_score >= -10 AND feedback_score <= 10),
    feedback_text TEXT,
    learned_pattern TEXT,
    created_by VARCHAR(100) DEFAULT 'system',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Technical debt metrics table
CREATE TABLE IF NOT EXISTS debt_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    measured_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    total_debt_score DECIMAL(10,2),
    debt_by_category JSONB DEFAULT '{}'::JSONB,
    debt_by_scenario JSONB DEFAULT '{}'::JSONB,
    debt_trend VARCHAR(20) CHECK (debt_trend IN ('increasing', 'stable', 'decreasing')),
    critical_issues INTEGER DEFAULT 0,
    high_issues INTEGER DEFAULT 0,
    medium_issues INTEGER DEFAULT 0,
    low_issues INTEGER DEFAULT 0,
    estimated_cleanup_hours DECIMAL(10,2),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Indexes for performance
CREATE INDEX idx_scan_history_status ON scan_history(status);
CREATE INDEX idx_scan_history_created ON scan_history(started_at DESC);
CREATE INDEX idx_scan_results_scan_id ON scan_results(scan_id);
CREATE INDEX idx_scan_results_severity ON scan_results(severity);
CREATE INDEX idx_scan_results_issue_type ON scan_results(issue_type);
CREATE INDEX idx_scan_results_file_path ON scan_results(file_path);
CREATE INDEX idx_cleanup_actions_result_id ON cleanup_actions(scan_result_id);
CREATE INDEX idx_cleanup_actions_executed ON cleanup_actions(executed_at DESC);
CREATE INDEX idx_cleanup_rules_enabled ON cleanup_rules(enabled) WHERE enabled = true;
CREATE INDEX idx_cleanup_rules_category ON cleanup_rules(category);
CREATE INDEX idx_learning_feedback_rule ON learning_feedback(cleanup_rule_id);
CREATE INDEX idx_debt_metrics_measured ON debt_metrics(measured_at DESC);

-- Functions for analytics
CREATE OR REPLACE FUNCTION tidiness.calculate_debt_score()
RETURNS DECIMAL AS $$
DECLARE
    score DECIMAL := 0;
BEGIN
    SELECT 
        SUM(CASE 
            WHEN severity = 'critical' THEN 10
            WHEN severity = 'high' THEN 5
            WHEN severity = 'medium' THEN 2
            WHEN severity = 'low' THEN 1
            ELSE 0
        END)
    INTO score
    FROM scan_results
    WHERE scan_id = (
        SELECT scan_id 
        FROM scan_history 
        WHERE status = 'completed' 
        ORDER BY completed_at DESC 
        LIMIT 1
    );
    
    RETURN COALESCE(score, 0);
END;
$$ LANGUAGE plpgsql;

-- Function to get cleanup success rate
CREATE OR REPLACE FUNCTION tidiness.get_cleanup_success_rate(days INTEGER DEFAULT 30)
RETURNS TABLE(
    total_cleanups BIGINT,
    successful_cleanups BIGINT,
    success_rate DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_cleanups,
        COUNT(*) FILTER (WHERE success = true) as successful_cleanups,
        ROUND(
            100.0 * COUNT(*) FILTER (WHERE success = true) / NULLIF(COUNT(*), 0), 
            2
        ) as success_rate
    FROM cleanup_actions
    WHERE executed_at >= CURRENT_TIMESTAMP - INTERVAL '1 day' * days;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION tidiness.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_cleanup_rules_timestamp
    BEFORE UPDATE ON cleanup_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Views for common queries
CREATE OR REPLACE VIEW tidiness.latest_scan_summary AS
SELECT 
    sh.scan_id,
    sh.started_at,
    sh.completed_at,
    sh.total_files_scanned,
    sh.total_issues_found,
    COUNT(DISTINCT sr.file_path) as unique_files_with_issues,
    COUNT(sr.id) FILTER (WHERE sr.severity = 'critical') as critical_count,
    COUNT(sr.id) FILTER (WHERE sr.severity = 'high') as high_count,
    COUNT(sr.id) FILTER (WHERE sr.severity = 'medium') as medium_count,
    COUNT(sr.id) FILTER (WHERE sr.severity = 'low') as low_count,
    COUNT(sr.id) FILTER (WHERE sr.safe_to_automate = true) as automatable_count
FROM scan_history sh
LEFT JOIN scan_results sr ON sh.scan_id = sr.scan_id
WHERE sh.status = 'completed'
GROUP BY sh.scan_id, sh.started_at, sh.completed_at, sh.total_files_scanned, sh.total_issues_found
ORDER BY sh.completed_at DESC
LIMIT 1;

CREATE OR REPLACE VIEW tidiness.cleanup_effectiveness AS
SELECT 
    cr.rule_name,
    cr.category,
    COUNT(ca.id) as times_applied,
    COUNT(ca.id) FILTER (WHERE ca.action_taken = 'accepted') as times_accepted,
    COUNT(ca.id) FILTER (WHERE ca.action_taken = 'rejected') as times_rejected,
    ROUND(
        100.0 * COUNT(ca.id) FILTER (WHERE ca.action_taken = 'accepted') / 
        NULLIF(COUNT(ca.id), 0), 
        2
    ) as acceptance_rate,
    SUM(ca.bytes_freed) as total_bytes_freed
FROM cleanup_rules cr
LEFT JOIN scan_results sr ON sr.issue_type = cr.rule_name
LEFT JOIN cleanup_actions ca ON ca.scan_result_id = sr.id
GROUP BY cr.id, cr.rule_name, cr.category
ORDER BY times_applied DESC;

-- Grant permissions (adjust as needed)
GRANT ALL ON SCHEMA tidiness TO postgres;
GRANT SELECT ON ALL TABLES IN SCHEMA tidiness TO PUBLIC;
GRANT INSERT, UPDATE ON ALL TABLES IN SCHEMA tidiness TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA tidiness TO PUBLIC;