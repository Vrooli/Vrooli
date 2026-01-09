-- Migration: Add missing columns to campaigns table
-- Date: 2025-10-03
-- Description: Adds icon and parent_id columns that were missing from initial deployment

-- Add icon column if it doesn't exist
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS icon VARCHAR(50) DEFAULT 'folder';

-- Add parent_id column if it doesn't exist (with foreign key reference)
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES campaigns(id) ON DELETE CASCADE;

-- Add index for parent_id if column was just created
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_campaigns_parent') THEN
        CREATE INDEX idx_campaigns_parent ON campaigns(parent_id);
    END IF;
END
$$;
