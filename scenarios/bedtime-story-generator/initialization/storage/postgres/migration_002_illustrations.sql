-- Migration to add illustrations support
-- Adds illustrations column to stories table to store emoji-based or future image illustrations

-- Add illustrations column to stories table
ALTER TABLE stories 
ADD COLUMN IF NOT EXISTS illustrations JSONB DEFAULT '{}'::jsonb;

-- Index for searching stories with illustrations
CREATE INDEX IF NOT EXISTS idx_stories_illustrations ON stories USING gin (illustrations);

-- Update metadata to track migration
INSERT INTO user_preferences (id, metadata)
VALUES (gen_random_uuid(), '{"migration": "002_illustrations", "applied_at": "' || CURRENT_TIMESTAMP || '"}')
ON CONFLICT (id) DO NOTHING;