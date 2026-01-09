-- Migration: Sync tech_trees table with current schema
-- Adds missing columns that were in schema.sql but not in the database

-- Add slug column
ALTER TABLE tech_trees
ADD COLUMN IF NOT EXISTS slug VARCHAR(128);

-- Add tree_type column
ALTER TABLE tech_trees
ADD COLUMN IF NOT EXISTS tree_type VARCHAR(50) DEFAULT 'official';

-- Add status column
ALTER TABLE tech_trees
ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active';

-- Add parent_tree_id column
ALTER TABLE tech_trees
ADD COLUMN IF NOT EXISTS parent_tree_id UUID REFERENCES tech_trees(id) ON DELETE SET NULL;

-- Backfill slug for existing trees (use name converted to slug format)
UPDATE tech_trees
SET slug = LOWER(REGEXP_REPLACE(REGEXP_REPLACE(TRIM(name), '[^a-zA-Z0-9-]+', '-', 'g'), '-+', '-', 'g'))
WHERE slug IS NULL;

-- Backfill tree_type for existing trees
UPDATE tech_trees
SET tree_type = 'official'
WHERE tree_type IS NULL;

-- Backfill status for existing trees
UPDATE tech_trees
SET status = 'active'
WHERE status IS NULL;

-- Make columns NOT NULL after backfilling
ALTER TABLE tech_trees
ALTER COLUMN slug SET NOT NULL;

ALTER TABLE tech_trees
ALTER COLUMN tree_type SET NOT NULL;

ALTER TABLE tech_trees
ALTER COLUMN status SET NOT NULL;

-- Add unique constraint on slug
ALTER TABLE tech_trees
DROP CONSTRAINT IF EXISTS tech_trees_slug_key;
ALTER TABLE tech_trees
ADD CONSTRAINT tech_trees_slug_key UNIQUE (slug);

-- Add index for slug lookups
CREATE INDEX IF NOT EXISTS idx_tech_trees_slug ON tech_trees(slug);

-- Add index for tree_type and status lookups
CREATE INDEX IF NOT EXISTS idx_tech_trees_type_status ON tech_trees(tree_type, status);
