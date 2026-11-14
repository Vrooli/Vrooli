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
-- wire format. The router holds NO provider documents and NO vectors here.
-- It owns this registry, query telemetry, and a provider-profile lineage feed
-- used to rebuild its own route-description index. The lineage feed contains
-- only fields already present in ProviderDescriptor.routing_profile and
-- never copies provider-owned corpus documents.
-- control_token is a retired legacy column. New and migrated bearer tokens
-- live only in the credential authority; Search Hub stores per-provider
-- witnesses below so loss of the authority value fails closed.
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

-- Non-secret credential-loss witnesses. A row means a provider has received a
-- token before, so an absent token-set credential cannot be mistaken for a
-- first registration and silently replaced.
CREATE TABLE IF NOT EXISTS provider_control_token_witness (
  provider_id TEXT PRIMARY KEY,
  minted_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_providers_group  ON providers(provider_group);
CREATE INDEX IF NOT EXISTS idx_providers_bucket ON providers(bucket);
CREATE INDEX IF NOT EXISTS idx_providers_type   ON providers(type);
CREATE INDEX IF NOT EXISTS idx_providers_state  ON providers(state);

-- Search Hub's provider-description embedding source is derived from these
-- authoritative descriptors. Keep a durable owner change feed so a pinned
-- generation can apply updates and tombstones without inferring history from
-- updated_at or trusting a caller-provided cursor.
CREATE TABLE IF NOT EXISTS provider_description_changes (
  sequence          INTEGER PRIMARY KEY AUTOINCREMENT,
  provider_id       TEXT NOT NULL,
  before_profile    TEXT,
  after_profile     TEXT,
  CHECK (before_profile IS NOT NULL OR after_profile IS NOT NULL)
);

CREATE TRIGGER IF NOT EXISTS provider_description_changes_insert
AFTER INSERT ON providers
WHEN NEW.state = 1 AND json_extract(NEW.descriptor, '$.endpoint') IS NOT NULL
BEGIN
  INSERT INTO provider_description_changes(provider_id, after_profile)
  SELECT NEW.provider_id,
    CASE WHEN NEW.state = 1 AND json_extract(NEW.descriptor, '$.endpoint') IS NOT NULL THEN json_object(
      'ProviderID', json_extract(NEW.descriptor, '$.provider_id'),
      'Type', json_extract(NEW.descriptor, '$.type'),
      'Group', json_extract(NEW.descriptor, '$.provider_group'),
      'Description', json_extract(NEW.descriptor, '$.description'),
      'AnswerSpaces', json(COALESCE(json_extract(NEW.descriptor, '$.routing_profile.answer_spaces'), '[]')),
      'Intents', json(COALESCE(json_extract(NEW.descriptor, '$.routing_profile.intents'), '[]')),
      'PositiveExamples', json(COALESCE(json_extract(NEW.descriptor, '$.routing_profile.positive_examples'), '[]')),
      'Exclusions', json(COALESCE(json_extract(NEW.descriptor, '$.routing_profile.exclusions'), '[]'))
    ) END;
END;

CREATE TRIGGER IF NOT EXISTS provider_description_changes_update
AFTER UPDATE OF descriptor, state ON providers
WHEN (OLD.descriptor <> NEW.descriptor OR OLD.state <> NEW.state)
 AND ((OLD.state = 1 AND json_extract(OLD.descriptor, '$.endpoint') IS NOT NULL)
   OR (NEW.state = 1 AND json_extract(NEW.descriptor, '$.endpoint') IS NOT NULL))
BEGIN
  INSERT INTO provider_description_changes(provider_id, before_profile, after_profile)
  SELECT NEW.provider_id,
    CASE WHEN OLD.state = 1 AND json_extract(OLD.descriptor, '$.endpoint') IS NOT NULL THEN json_object(
      'ProviderID', json_extract(OLD.descriptor, '$.provider_id'),
      'Type', json_extract(OLD.descriptor, '$.type'),
      'Group', json_extract(OLD.descriptor, '$.provider_group'),
      'Description', json_extract(OLD.descriptor, '$.description'),
      'AnswerSpaces', json(COALESCE(json_extract(OLD.descriptor, '$.routing_profile.answer_spaces'), '[]')),
      'Intents', json(COALESCE(json_extract(OLD.descriptor, '$.routing_profile.intents'), '[]')),
      'PositiveExamples', json(COALESCE(json_extract(OLD.descriptor, '$.routing_profile.positive_examples'), '[]')),
      'Exclusions', json(COALESCE(json_extract(OLD.descriptor, '$.routing_profile.exclusions'), '[]'))
    ) END,
    CASE WHEN NEW.state = 1 AND json_extract(NEW.descriptor, '$.endpoint') IS NOT NULL THEN json_object(
      'ProviderID', json_extract(NEW.descriptor, '$.provider_id'),
      'Type', json_extract(NEW.descriptor, '$.type'),
      'Group', json_extract(NEW.descriptor, '$.provider_group'),
      'Description', json_extract(NEW.descriptor, '$.description'),
      'AnswerSpaces', json(COALESCE(json_extract(NEW.descriptor, '$.routing_profile.answer_spaces'), '[]')),
      'Intents', json(COALESCE(json_extract(NEW.descriptor, '$.routing_profile.intents'), '[]')),
      'PositiveExamples', json(COALESCE(json_extract(NEW.descriptor, '$.routing_profile.positive_examples'), '[]')),
      'Exclusions', json(COALESCE(json_extract(NEW.descriptor, '$.routing_profile.exclusions'), '[]'))
    ) END;
END;

CREATE TRIGGER IF NOT EXISTS provider_description_changes_delete
AFTER DELETE ON providers
WHEN OLD.state = 1 AND json_extract(OLD.descriptor, '$.endpoint') IS NOT NULL
BEGIN
  INSERT INTO provider_description_changes(provider_id, before_profile)
  SELECT OLD.provider_id,
    CASE WHEN OLD.state = 1 AND json_extract(OLD.descriptor, '$.endpoint') IS NOT NULL THEN json_object(
      'ProviderID', json_extract(OLD.descriptor, '$.provider_id'),
      'Type', json_extract(OLD.descriptor, '$.type'),
      'Group', json_extract(OLD.descriptor, '$.provider_group'),
      'Description', json_extract(OLD.descriptor, '$.description'),
      'AnswerSpaces', json(COALESCE(json_extract(OLD.descriptor, '$.routing_profile.answer_spaces'), '[]')),
      'Intents', json(COALESCE(json_extract(OLD.descriptor, '$.routing_profile.intents'), '[]')),
      'PositiveExamples', json(COALESCE(json_extract(OLD.descriptor, '$.routing_profile.positive_examples'), '[]')),
      'Exclusions', json(COALESCE(json_extract(OLD.descriptor, '$.routing_profile.exclusions'), '[]'))
    ) END;
END;
