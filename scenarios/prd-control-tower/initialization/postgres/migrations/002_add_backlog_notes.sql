-- Add notes column to backlog entries so agents can attach context
ALTER TABLE IF EXISTS backlog_entries
    ADD COLUMN IF NOT EXISTS notes TEXT;
