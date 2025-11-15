-- Migration: Add slug column to tech_trees table
-- This enables URL-friendly identification and navigation of tech trees

-- Add slug column to tech_trees
ALTER TABLE tech_trees
ADD COLUMN IF NOT EXISTS slug VARCHAR(128);

-- Backfill slug for existing trees (use name converted to slug format)
UPDATE tech_trees
SET slug = LOWER(REGEXP_REPLACE(REGEXP_REPLACE(name, '[^a-zA-Z0-9-]+', '-', 'g'), '-+', '-', 'g'))
WHERE slug IS NULL;

-- Make slug NOT NULL after backfilling
ALTER TABLE tech_trees
ALTER COLUMN slug SET NOT NULL;

-- Add unique constraint
ALTER TABLE tech_trees
DROP CONSTRAINT IF EXISTS tech_trees_slug_key;
ALTER TABLE tech_trees
ADD CONSTRAINT tech_trees_slug_key UNIQUE (slug);

-- Add index for slug lookups
CREATE INDEX IF NOT EXISTS idx_tech_trees_slug ON tech_trees(slug);
