# Data — Audio Tools

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

The template default is embedded SQLite through `modernc.org/sqlite`.
The lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`, and
the API applies schemas on startup through `api-core/database`.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Encrypted BYOK credentials | byokstore | SQLite | `api/internal/byokstore/schema.sql` | Until deleted through Settings | Ciphertext only; encryption key is scenario runtime state. |
| Provider routing and voice overrides | settings | SQLite | `api/internal/settings/schema.sql` | Until changed or deleted through Settings | Local/provider configuration, never raw secrets. |
| STT configuration, wake words, and speaker profiles | stt | SQLite | `api/internal/stt/schema.sql` | Until user deletion | Speaker resource owns canonical embeddings; this stores metadata and bindings. |
| TTS configuration and playback events | tts | SQLite | `api/internal/tts/schema.sql` | Config until changed; events are local history | Playback events are idempotent by event ID. |
| Usage accounting | usagereport | SQLite | `api/internal/usagereport/schema.sql` | Local runtime history | Supports provider and fallback reporting. |
| Corpus and experiment bytes | corpus, experiment | Filesystem blob stores | Domain blob-store implementations | Explicit domain lifecycle | Mutable bytes are routed through `api-core/storage`. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| byok_credentials | byokstore | `api/internal/byokstore/schema.sql` | Settings credential repository |
| provider_config, voice_overrides | settings | `api/internal/settings/schema.sql` | Settings repository |
| usage_rows | usagereport | `api/internal/usagereport/schema.sql` | Usage recorder and reports |
| wakeword_templates, speaker_profiles, stt_stream_config, stt_speaker_config | stt | `api/internal/stt/schema.sql` | STT stores and handlers |
| tts_config, playback_events | tts | `api/internal/tts/schema.sql` | TTS stores and handlers |
| system schema | infrastructure | `api/internal/database/system.sql` | Cross-cutting infrastructure only; currently empty |

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| None yet. | n/a | n/a | Add when product requirements include import/export. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Encrypted BYOK credentials | Settings deletion | Local runtime data | Key rotation and broader credential retention policy remain an operator concern. |
| Audio configuration and profiles | Corresponding Settings/STT deletion operation | Local runtime data | Speaker resource data follows its resource lifecycle. |
| Usage and playback history | Local scenario data lifecycle | Local runtime history | No automatic retention policy is configured yet. |

## Privacy Notes

Generated template data is local development data. If a scenario stores
personal, regulated, customer, financial, or sensitive business data,
update this document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before implementation expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
