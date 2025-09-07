-- Ecosystem Manager Database Schema
-- Minimal schema focused on task history and operational metrics

-- Task execution history for analytics and learning
CREATE TABLE IF NOT EXISTS task_executions (
    id SERIAL PRIMARY KEY,
    task_id VARCHAR(255) NOT NULL,
    operation_type VARCHAR(50) NOT NULL, -- resource-generator, resource-improver, scenario-generator, scenario-improver
    target_type VARCHAR(20) NOT NULL,    -- resource, scenario  
    target_name VARCHAR(255),            -- name of resource/scenario being worked on
    category VARCHAR(50),                -- ai-ml, communication, etc.
    priority VARCHAR(20),                -- low, medium, high, critical
    effort_estimate VARCHAR(20),         -- 1h, 2h, 4h, 8h, 16h+
    
    -- Timing information
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_minutes INTEGER,
    
    -- Execution details
    status VARCHAR(20) NOT NULL,         -- completed, failed, timeout
    claude_code_used BOOLEAN DEFAULT true,
    error_details TEXT,
    success_details TEXT,
    
    -- PRD tracking (for scenarios)
    prd_completion_before INTEGER,       -- percentage before task
    prd_completion_after INTEGER,        -- percentage after task
    
    -- Task metadata
    created_by VARCHAR(100),
    tags TEXT[],                         -- array of tags
    
    -- Indexes for common queries
    CONSTRAINT task_executions_status_check CHECK (status IN ('completed', 'failed', 'timeout', 'cancelled'))
);

-- Operation metrics for success rate analysis
CREATE TABLE IF NOT EXISTS operation_metrics (
    id SERIAL PRIMARY KEY,
    operation_type VARCHAR(50) NOT NULL,
    target_type VARCHAR(20) NOT NULL,
    category VARCHAR(50),
    date_bucket DATE NOT NULL,           -- daily aggregation
    
    -- Success metrics
    total_attempts INTEGER DEFAULT 0,
    successful_completions INTEGER DEFAULT 0,
    failures INTEGER DEFAULT 0,
    timeouts INTEGER DEFAULT 0,
    
    -- Timing metrics
    avg_duration_minutes INTEGER,
    min_duration_minutes INTEGER,
    max_duration_minutes INTEGER,
    
    -- PRD improvement metrics (scenarios only)
    avg_prd_improvement INTEGER,         -- average percentage point improvement
    total_prd_points_improved INTEGER,
    
    -- Common patterns
    common_failure_reasons TEXT[],
    common_success_patterns TEXT[],
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint to prevent duplicates
    UNIQUE(operation_type, target_type, category, date_bucket)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_task_executions_operation_type ON task_executions(operation_type);
CREATE INDEX IF NOT EXISTS idx_task_executions_target_name ON task_executions(target_name);
CREATE INDEX IF NOT EXISTS idx_task_executions_started_at ON task_executions(started_at);
CREATE INDEX IF NOT EXISTS idx_task_executions_status ON task_executions(status);

CREATE INDEX IF NOT EXISTS idx_operation_metrics_date ON operation_metrics(date_bucket);
CREATE INDEX IF NOT EXISTS idx_operation_metrics_operation ON operation_metrics(operation_type);

-- Function to update metrics when task executions are inserted
CREATE OR REPLACE FUNCTION update_operation_metrics()
RETURNS TRIGGER AS $$
BEGIN
    -- Only update metrics when a task is completed (has completed_at timestamp)
    IF NEW.completed_at IS NOT NULL AND (OLD.completed_at IS NULL OR OLD IS NULL) THEN
        
        -- Calculate duration
        NEW.duration_minutes := EXTRACT(EPOCH FROM (NEW.completed_at - NEW.started_at)) / 60;
        
        -- Update daily metrics
        INSERT INTO operation_metrics (
            operation_type, target_type, category, date_bucket,
            total_attempts, successful_completions, failures, timeouts,
            avg_duration_minutes, min_duration_minutes, max_duration_minutes,
            avg_prd_improvement, total_prd_points_improved
        ) VALUES (
            NEW.operation_type, NEW.target_type, NEW.category, DATE(NEW.completed_at),
            1,
            CASE WHEN NEW.status = 'completed' THEN 1 ELSE 0 END,
            CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
            CASE WHEN NEW.status = 'timeout' THEN 1 ELSE 0 END,
            NEW.duration_minutes,
            NEW.duration_minutes,
            NEW.duration_minutes,
            COALESCE(NEW.prd_completion_after - NEW.prd_completion_before, 0),
            COALESCE(NEW.prd_completion_after - NEW.prd_completion_before, 0)
        )
        ON CONFLICT (operation_type, target_type, category, date_bucket)
        DO UPDATE SET
            total_attempts = operation_metrics.total_attempts + 1,
            successful_completions = operation_metrics.successful_completions + 
                CASE WHEN NEW.status = 'completed' THEN 1 ELSE 0 END,
            failures = operation_metrics.failures + 
                CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
            timeouts = operation_metrics.timeouts + 
                CASE WHEN NEW.status = 'timeout' THEN 1 ELSE 0 END,
            avg_duration_minutes = (
                operation_metrics.avg_duration_minutes * operation_metrics.total_attempts + NEW.duration_minutes
            ) / (operation_metrics.total_attempts + 1),
            min_duration_minutes = LEAST(operation_metrics.min_duration_minutes, NEW.duration_minutes),
            max_duration_minutes = GREATEST(operation_metrics.max_duration_minutes, NEW.duration_minutes),
            avg_prd_improvement = (
                operation_metrics.avg_prd_improvement * operation_metrics.total_attempts + 
                COALESCE(NEW.prd_completion_after - NEW.prd_completion_before, 0)
            ) / (operation_metrics.total_attempts + 1),
            total_prd_points_improved = operation_metrics.total_prd_points_improved + 
                COALESCE(NEW.prd_completion_after - NEW.prd_completion_before, 0),
            updated_at = CURRENT_TIMESTAMP;
            
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update metrics
CREATE TRIGGER trigger_update_operation_metrics
    BEFORE INSERT OR UPDATE ON task_executions
    FOR EACH ROW
    EXECUTE FUNCTION update_operation_metrics();

-- Views for common queries
CREATE OR REPLACE VIEW task_execution_summary AS
SELECT 
    operation_type,
    target_type,
    category,
    COUNT(*) as total_tasks,
    COUNT(*) FILTER (WHERE status = 'completed') as successful_tasks,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_tasks,
    ROUND(AVG(duration_minutes), 1) as avg_duration_minutes,
    ROUND(AVG(prd_completion_after - prd_completion_before) FILTER (WHERE prd_completion_after IS NOT NULL), 1) as avg_prd_improvement
FROM task_executions 
WHERE completed_at IS NOT NULL
GROUP BY operation_type, target_type, category
ORDER BY total_tasks DESC;

CREATE OR REPLACE VIEW recent_task_activity AS
SELECT 
    task_id,
    operation_type || ' → ' || COALESCE(target_name, 'new ' || target_type) as description,
    status,
    duration_minutes,
    started_at,
    completed_at
FROM task_executions 
WHERE started_at > CURRENT_TIMESTAMP - INTERVAL '7 days'
ORDER BY started_at DESC
LIMIT 50;

-- Sample data for testing (optional)
-- INSERT INTO task_executions (task_id, operation_type, target_type, target_name, category, status, completed_at)
-- VALUES ('test-001', 'resource-generator', 'resource', 'test-matrix', 'communication', 'completed', CURRENT_TIMESTAMP);