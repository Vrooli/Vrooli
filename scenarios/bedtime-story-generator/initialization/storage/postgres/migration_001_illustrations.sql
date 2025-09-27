-- Migration to add illustrations column
-- Adds support for storing emoji-based illustrations per story

ALTER TABLE stories 
ADD COLUMN IF NOT EXISTS illustrations JSONB DEFAULT '{}'::jsonb;

-- Add index for faster queries on stories with illustrations
CREATE INDEX IF NOT EXISTS idx_stories_has_illustrations 
ON stories((illustrations != '{}'::jsonb));

-- Comment on new column
COMMENT ON COLUMN stories.illustrations IS 'Emoji-based illustrations for each story page';