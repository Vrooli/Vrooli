-- Campaign tables — owned by internal/campaign/. Persist the stateful
-- scenario-improvement tracker: one row per campaign, one row per tracked
-- item (finding). Items are source-agnostic (the shared ArchitectureFinding
-- contract); the reconciliation key is the afid stable_id.
--
-- An item's identity inside a campaign is (campaign_id, stable_id):
-- re-audits match purely on stable_id, so the same defect collapses onto
-- the same row across audits and the lifecycle state survives.

CREATE TABLE IF NOT EXISTS campaigns (
  id          TEXT PRIMARY KEY,
  scenario    TEXT NOT NULL,
  name        TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_campaigns_scenario
  ON campaigns(scenario, created_at DESC);

CREATE TABLE IF NOT EXISTS campaign_items (
  campaign_id     TEXT NOT NULL,
  stable_id       TEXT NOT NULL,
  scenario        TEXT NOT NULL,
  source          TEXT NOT NULL,
  code            TEXT NOT NULL,
  severity        TEXT NOT NULL,
  locations       TEXT NOT NULL DEFAULT '[]',
  domains         TEXT NOT NULL DEFAULT '[]',
  message         TEXT NOT NULL DEFAULT '',
  suggestion      TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL,
  resolution_note TEXT NOT NULL DEFAULT '',
  regressed       INTEGER NOT NULL DEFAULT 0,
  effort          TEXT NOT NULL DEFAULT 'unspecified',
  first_seen_at   TEXT NOT NULL,
  updated_at      TEXT NOT NULL,
  PRIMARY KEY (campaign_id, stable_id),
  FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_campaign_items_status
  ON campaign_items(campaign_id, status);
