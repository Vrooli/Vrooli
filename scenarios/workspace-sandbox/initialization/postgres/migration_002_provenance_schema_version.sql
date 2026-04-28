-- Adds the auditability-contract provenance fields to applied_changes.

-- All workspace-sandbox tables live in the workspace_sandbox schema (created
-- by migration_001). Pin search_path so this migration applies there
-- regardless of the bootstrap context's default schema.
SET search_path TO workspace_sandbox, public;

-- These fields are part of the shared sandbox-provenance schema (v1.0.0)
-- coordinated with execute/gct-pending-ai-provenance-hardening. See
-- packages/sandbox-provenance/COORDINATION.md.
--
-- Field semantics (sandbox-provenance schema 1.0.0):
--   provenance_state ∈ {applied, pending-review, denied}
--   run_outcome      ∈ {success, failure, cancelled, timeout}
--   conversation_id  groups runs that belong to the same agent thread
--   cost_usd         total USD cost of the originating run
--   schema_version   pinned to "1.0.0" so readers can fail loud on drift

ALTER TABLE applied_changes ADD COLUMN IF NOT EXISTS provenance_state TEXT;
ALTER TABLE applied_changes ADD COLUMN IF NOT EXISTS run_outcome TEXT;
ALTER TABLE applied_changes ADD COLUMN IF NOT EXISTS conversation_id TEXT;
ALTER TABLE applied_changes ADD COLUMN IF NOT EXISTS cost_usd DOUBLE PRECISION;
ALTER TABLE applied_changes ADD COLUMN IF NOT EXISTS schema_version TEXT;

-- Index on conversation_id for cross-run grouping (GCT and web-console both
-- group provenance by conversation).
CREATE INDEX IF NOT EXISTS idx_applied_changes_conversation_id
  ON applied_changes(conversation_id)
  WHERE conversation_id IS NOT NULL;
