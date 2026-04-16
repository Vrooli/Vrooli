-- Migration: Rename prompts to skills, avatars to members
-- Date: 2026-01-23
-- Description: Renames prompt_metrics table to skill_metrics and updates column names

-- Rename table prompt_metrics to skill_metrics
DO $$ BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'prompt_metrics'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'skill_metrics'
    ) THEN
        ALTER TABLE prompt_metrics RENAME TO skill_metrics;
    END IF;
END $$;

-- Rename column prompt_id to skill_id in skill_metrics
DO $$ BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'skill_metrics'
          AND column_name = 'prompt_id'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'skill_metrics'
          AND column_name = 'skill_id'
    ) THEN
        ALTER TABLE skill_metrics RENAME COLUMN prompt_id TO skill_id;
    END IF;
END $$;

-- Rename column prompt_id to skill_id in test_results
DO $$ BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'test_results'
          AND column_name = 'prompt_id'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'test_results'
          AND column_name = 'skill_id'
    ) THEN
        ALTER TABLE test_results RENAME COLUMN prompt_id TO skill_id;
    END IF;
END $$;

-- Drop old indexes and create new ones with updated names
DROP INDEX IF EXISTS idx_prompt_metrics_prompt_id;
DROP INDEX IF EXISTS idx_prompt_metrics_usage;
DROP INDEX IF EXISTS idx_prompt_metrics_last_used;
DROP INDEX IF EXISTS idx_test_results_prompt;

CREATE INDEX IF NOT EXISTS idx_skill_metrics_skill_id ON skill_metrics(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_metrics_usage ON skill_metrics(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_skill_metrics_last_used ON skill_metrics(last_used DESC);
CREATE INDEX IF NOT EXISTS idx_test_results_skill ON test_results(skill_id);

-- Drop old trigger and function, recreate with new names
DROP TRIGGER IF EXISTS trigger_update_prompt_metrics_timestamp ON skill_metrics;
DROP FUNCTION IF EXISTS update_prompt_metrics_timestamp();

-- Create new function with updated name
CREATE OR REPLACE FUNCTION update_skill_metrics_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create new trigger
DROP TRIGGER IF EXISTS trigger_update_skill_metrics_timestamp ON skill_metrics;
CREATE TRIGGER trigger_update_skill_metrics_timestamp
    BEFORE UPDATE ON skill_metrics
    FOR EACH ROW EXECUTE FUNCTION update_skill_metrics_timestamp();
