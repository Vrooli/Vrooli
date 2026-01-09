-- Migration 001: Expand file_metrics to store detailed code quality metrics
-- This enables fast refactor recommendations without re-scanning files

-- Add detailed code metrics columns to file_metrics table
ALTER TABLE file_metrics
ADD COLUMN IF NOT EXISTS language VARCHAR(20),              -- go, typescript, javascript, python, rust
ADD COLUMN IF NOT EXISTS todo_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS fixme_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS hack_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS import_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS function_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS code_lines INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS comment_lines INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS comment_to_code_ratio FLOAT DEFAULT 0.0,
ADD COLUMN IF NOT EXISTS has_test_file BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS complexity_avg FLOAT,              -- nullable - not all languages supported
ADD COLUMN IF NOT EXISTS complexity_max INTEGER,            -- nullable
ADD COLUMN IF NOT EXISTS duplication_pct FLOAT,             -- nullable - requires external tools
ADD COLUMN IF NOT EXISTS file_extension VARCHAR(10);

-- Add index for language-based queries
CREATE INDEX IF NOT EXISTS idx_file_metrics_language ON file_metrics(language);

-- Add index for filtering by code quality metrics
CREATE INDEX IF NOT EXISTS idx_file_metrics_line_count ON file_metrics(line_count DESC);
CREATE INDEX IF NOT EXISTS idx_file_metrics_complexity ON file_metrics(complexity_max DESC NULLS LAST);
