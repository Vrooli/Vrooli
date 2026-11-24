CREATE TABLE IF NOT EXISTS provider_setup_steps (
  provider_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  ask_id TEXT NOT NULL,
  event_time TEXT NOT NULL,
  state TEXT NOT NULL,
  answer TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  PRIMARY KEY(provider_id, step_id),
  UNIQUE(ask_id)
);

CREATE TABLE IF NOT EXISTS provider_setup_evidence (
  provider_id TEXT NOT NULL,
  evidence_id TEXT NOT NULL,
  state TEXT NOT NULL,
  verified INTEGER NOT NULL,
  detail TEXT NOT NULL,
  checked_at TEXT NOT NULL,
  source_url TEXT NOT NULL DEFAULT '',
  source_version TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(provider_id, evidence_id)
);

CREATE TABLE IF NOT EXISTS provider_setup_prices (
  provider_id TEXT NOT NULL,
  region TEXT NOT NULL,
  size TEXT NOT NULL,
  offer_id TEXT NOT NULL DEFAULT '',
  currency TEXT NOT NULL,
  server_minor INTEGER NOT NULL,
  ipv4_minor INTEGER NOT NULL,
  tax_basis TEXT NOT NULL,
  source_url TEXT NOT NULL,
  source_version TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  provenance_json TEXT NOT NULL DEFAULT '',
  provenance_fingerprint TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(provider_id, region, size)
);
