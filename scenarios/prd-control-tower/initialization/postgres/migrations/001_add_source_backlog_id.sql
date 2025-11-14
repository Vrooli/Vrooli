-- Migration: Add source_backlog_id to drafts table
-- Date: 2025-11-14
-- Purpose: Enable backlog → draft traceability

-- Add column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'drafts' AND column_name = 'source_backlog_id'
    ) THEN
        ALTER TABLE drafts ADD COLUMN source_backlog_id UUID NULL;
    END IF;
END $$;

-- Optionally create index for reverse lookups (backlog -> draft)
CREATE INDEX IF NOT EXISTS idx_drafts_backlog_source ON drafts(source_backlog_id) WHERE source_backlog_id IS NOT NULL;

-- Backfill: Link existing drafts to their backlog entries (if converted_draft_id exists)
UPDATE drafts d
SET source_backlog_id = b.id
FROM backlog_entries b
WHERE b.converted_draft_id = d.id
  AND d.source_backlog_id IS NULL;
