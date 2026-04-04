-- Add git_commit_hash to visual_validations for linking to deployment approvals.
-- Nullable so legacy rows get NULL (no bridging for those).
ALTER TABLE visual_validations ADD COLUMN IF NOT EXISTS git_commit_hash TEXT;
