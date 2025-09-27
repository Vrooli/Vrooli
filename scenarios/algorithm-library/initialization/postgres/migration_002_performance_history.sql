-- Migration: Add performance history tracking
-- This migration adds tables to track performance trends over time

-- Performance history table - tracks algorithm performance over time
CREATE TABLE IF NOT EXISTS performance_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm_id UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    implementation_id UUID REFERENCES implementations(id) ON DELETE CASCADE,
    language programming_language NOT NULL,
    
    -- Performance metrics
    avg_execution_time_ms DECIMAL(10,2) NOT NULL,
    min_execution_time_ms DECIMAL(10,2) NOT NULL,
    max_execution_time_ms DECIMAL(10,2) NOT NULL,
    std_dev_time_ms DECIMAL(10,2),
    
    avg_memory_mb DECIMAL(10,2) NOT NULL,
    min_memory_mb DECIMAL(10,2) NOT NULL,
    max_memory_mb DECIMAL(10,2) NOT NULL,
    
    -- Test conditions
    input_size INTEGER NOT NULL,
    sample_count INTEGER NOT NULL DEFAULT 1,
    test_category VARCHAR(50), -- 'small', 'medium', 'large', 'edge_case'
    
    -- Comparison metrics
    performance_score DECIMAL(5,2), -- 0-100 score relative to other implementations
    rank_in_category INTEGER, -- Rank among same algorithm implementations
    
    -- Metadata
    environment_info JSONB,
    notes TEXT,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure we don't duplicate entries
    UNIQUE(algorithm_id, implementation_id, language, input_size, recorded_at)
);

-- Create indexes for efficient querying
CREATE INDEX idx_perf_history_algorithm ON performance_history(algorithm_id);
CREATE INDEX idx_perf_history_implementation ON performance_history(implementation_id);
CREATE INDEX idx_perf_history_language ON performance_history(language);
CREATE INDEX idx_perf_history_recorded ON performance_history(recorded_at);
CREATE INDEX idx_perf_history_input_size ON performance_history(input_size);

-- Performance trends view - aggregated weekly performance data
CREATE OR REPLACE VIEW performance_trends AS
SELECT 
    ph.algorithm_id,
    a.name as algorithm_name,
    a.display_name,
    ph.language,
    DATE_TRUNC('week', ph.recorded_at) as week,
    AVG(ph.avg_execution_time_ms) as avg_weekly_time_ms,
    AVG(ph.avg_memory_mb) as avg_weekly_memory_mb,
    AVG(ph.performance_score) as avg_weekly_score,
    COUNT(*) as sample_count,
    MIN(ph.recorded_at) as first_recorded,
    MAX(ph.recorded_at) as last_recorded
FROM performance_history ph
JOIN algorithms a ON a.id = ph.algorithm_id
GROUP BY 
    ph.algorithm_id,
    a.name,
    a.display_name,
    ph.language,
    DATE_TRUNC('week', ph.recorded_at)
ORDER BY week DESC;

-- Performance comparison table - track head-to-head comparisons
CREATE TABLE IF NOT EXISTS performance_comparisons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    comparison_name VARCHAR(255) NOT NULL,
    
    -- Algorithms being compared
    algorithm_ids UUID[] NOT NULL,
    implementation_ids UUID[],
    languages programming_language[],
    
    -- Test parameters
    input_sizes INTEGER[] NOT NULL,
    test_data_description TEXT,
    
    -- Results
    results JSONB NOT NULL, -- Detailed comparison results
    winner_id UUID REFERENCES algorithms(id),
    summary TEXT,
    
    -- Metadata
    compared_by VARCHAR(100),
    compared_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_perf_comparisons_date ON performance_comparisons(compared_at);
CREATE INDEX idx_perf_comparisons_algorithms ON performance_comparisons USING GIN(algorithm_ids);

-- Function to calculate performance score relative to other implementations
CREATE OR REPLACE FUNCTION calculate_performance_score(
    p_algorithm_id UUID,
    p_execution_time_ms DECIMAL,
    p_memory_mb DECIMAL
) RETURNS DECIMAL AS $$
DECLARE
    v_score DECIMAL;
    v_time_rank INTEGER;
    v_memory_rank INTEGER;
    v_total_implementations INTEGER;
BEGIN
    -- Get ranking for execution time (lower is better)
    SELECT COUNT(*) + 1 INTO v_time_rank
    FROM performance_history
    WHERE algorithm_id = p_algorithm_id
    AND avg_execution_time_ms < p_execution_time_ms;
    
    -- Get ranking for memory usage (lower is better)
    SELECT COUNT(*) + 1 INTO v_memory_rank
    FROM performance_history
    WHERE algorithm_id = p_algorithm_id
    AND avg_memory_mb < p_memory_mb;
    
    -- Get total number of implementations
    SELECT COUNT(DISTINCT implementation_id) INTO v_total_implementations
    FROM performance_history
    WHERE algorithm_id = p_algorithm_id;
    
    -- Calculate score (weighted: 60% time, 40% memory)
    IF v_total_implementations > 0 THEN
        v_score := 100.0 * (
            0.6 * (1.0 - (v_time_rank::DECIMAL / v_total_implementations)) +
            0.4 * (1.0 - (v_memory_rank::DECIMAL / v_total_implementations))
        );
    ELSE
        v_score := 50.0; -- Default score if no comparisons available
    END IF;
    
    RETURN ROUND(v_score, 2);
END;
$$ LANGUAGE plpgsql;

-- Trigger to update performance scores when new history is added
CREATE OR REPLACE FUNCTION update_performance_scores() RETURNS TRIGGER AS $$
BEGIN
    -- Update the performance score for the new entry
    NEW.performance_score := calculate_performance_score(
        NEW.algorithm_id,
        NEW.avg_execution_time_ms,
        NEW.avg_memory_mb
    );
    
    -- Update implementation's latest performance score
    UPDATE implementations
    SET performance_score = NEW.performance_score,
        execution_time_ms = NEW.avg_execution_time_ms,
        memory_usage_bytes = NEW.avg_memory_mb * 1024 * 1024
    WHERE id = NEW.implementation_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_performance_scores
BEFORE INSERT OR UPDATE ON performance_history
FOR EACH ROW
EXECUTE FUNCTION update_performance_scores();

-- Add migration record (commented out since migrations table may not exist)
-- INSERT INTO migrations (name, applied_at) 
-- VALUES ('002_performance_history', NOW());