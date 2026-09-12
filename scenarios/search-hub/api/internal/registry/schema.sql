-- Registry table — owned by internal/registry/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot (api/main.go through the
-- modules.AllSchemas registry). Use CREATE TABLE IF NOT EXISTS so re-runs
-- are no-ops (forward-only declarative).
--
-- One row = one provider leaf (a (corpus, type) pair). The full
-- ProviderDescriptor is persisted as a protojson blob in `descriptor`; the
-- projected columns (provider_group, bucket, type, state, scope) exist only
-- so ListProviders can filter without parsing every blob. The blob is the
-- source of truth; the columns are a denormalized index kept in lockstep by
-- the store on every upsert.
--
-- Times are RFC3339Nano strings, matching the notes-domain convention and the
-- wire format. The router holds NO corpus content and NO vectors here — only
-- the registry (this table) and, from Phase 7, query telemetry.
-- control_token is the per-provider secret search-hub mints at first
-- registration and echoes on every subsequent register. It gates the dangerous
-- verbs (per-request query-time overrides, reindex, config-write): the provider
-- caches it in memory from the register response and validates incoming
-- override/reindex/config calls against it, while search-hub presents it when
-- it calls the provider. Stored server-side here so it survives a search-hub
-- restart (the provider re-acquires it via the next register echo). Empty until
-- the provider has registered at least once.
CREATE TABLE IF NOT EXISTS providers (
  provider_id    TEXT PRIMARY KEY,
  provider_group TEXT NOT NULL DEFAULT '',
  bucket         INTEGER NOT NULL DEFAULT 0,
  type           TEXT NOT NULL DEFAULT '',
  state          INTEGER NOT NULL DEFAULT 0,
  scope          INTEGER NOT NULL DEFAULT 0,
  descriptor     TEXT NOT NULL,
  control_token  TEXT NOT NULL DEFAULT '',
  created_at     TEXT NOT NULL,
  updated_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_providers_group  ON providers(provider_group);
CREATE INDEX IF NOT EXISTS idx_providers_bucket ON providers(bucket);
CREATE INDEX IF NOT EXISTS idx_providers_type   ON providers(type);
CREATE INDEX IF NOT EXISTS idx_providers_state  ON providers(state);
