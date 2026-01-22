-- Prompt Manager Database Schema v2.0
-- Simplified schema for file-based prompt storage with metrics tracking

-- Tags for categorizing prompts
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    color VARCHAR(7),
    description TEXT
);

-- Test results for prompt testing with LLMs
CREATE TABLE IF NOT EXISTS test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL DEFAULT 'ollama/llama3.2',
    input_variables JSONB,
    response TEXT,
    response_time FLOAT,
    token_count INTEGER,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    notes TEXT,
    tested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Prompt metrics for usage tracking
CREATE TABLE IF NOT EXISTS prompt_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id VARCHAR(100) NOT NULL UNIQUE,
    usage_count INTEGER DEFAULT 0,
    last_used TIMESTAMP,
    effectiveness_rating INTEGER CHECK (effectiveness_rating >= 1 AND effectiveness_rating <= 5),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
CREATE INDEX IF NOT EXISTS idx_test_results_prompt ON test_results(prompt_id);
CREATE INDEX IF NOT EXISTS idx_test_results_tested_at ON test_results(tested_at DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_metrics_prompt_id ON prompt_metrics(prompt_id);
CREATE INDEX IF NOT EXISTS idx_prompt_metrics_usage ON prompt_metrics(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_metrics_last_used ON prompt_metrics(last_used DESC);

-- Function to update prompt_metrics updated_at timestamp
CREATE OR REPLACE FUNCTION update_prompt_metrics_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at on prompt_metrics
DROP TRIGGER IF EXISTS trigger_update_prompt_metrics_timestamp ON prompt_metrics;
CREATE TRIGGER trigger_update_prompt_metrics_timestamp
    BEFORE UPDATE ON prompt_metrics
    FOR EACH ROW EXECUTE FUNCTION update_prompt_metrics_timestamp();
