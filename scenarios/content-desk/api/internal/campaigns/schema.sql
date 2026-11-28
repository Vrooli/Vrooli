CREATE TABLE IF NOT EXISTS campaigns (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  scenario_names TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS campaign_evidence_refs (
  campaign_id TEXT NOT NULL REFERENCES campaigns(id),
  ref TEXT NOT NULL,
  PRIMARY KEY (campaign_id, ref)
);

CREATE TABLE IF NOT EXISTS campaign_slots (
  campaign_id TEXT NOT NULL REFERENCES campaigns(id),
  channel TEXT NOT NULL,
  format TEXT NOT NULL,
  capacity INTEGER NOT NULL CHECK (capacity >= 0),
  reserved INTEGER NOT NULL DEFAULT 0 CHECK (reserved >= 0),
  PRIMARY KEY (campaign_id, channel, format),
  CHECK (reserved <= capacity)
);
