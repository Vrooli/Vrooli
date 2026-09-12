# Data

## Purpose Of This Document

This document records data ownership, storage shape, and import/export behavior.

## Storage Overview

PostgreSQL is the primary durable store. The UI can export local views to CSV/JSON. Redis is declared for future caching but is not directly used by current API code.

## Data Ownership

The API owns persisted data validation and writes. The UI owns transient view state and downloaded export files.

## Schema Map

The database schema is defined in [CODE: api/internal/<domain>/schema.sql]. It includes:

- `regions` for mapped foliage regions and typical peak weeks.
- `foliage_observations` for observed foliage percentage, color intensity, and peak status.
- `weather_data` for weather inputs used by prediction and status views.
- `foliage_predictions` for generated peak date predictions.
- `user_reports` for crowd-sourced reports and photo URLs.
- `trip_plans` for saved multi-region itineraries.

## Migrations And Compatibility

Schema initialization is owned by lifecycle resource bootstrap. Keep changes additive unless all tests and seed data are migrated together.

## Import / Export

The scenario imports seed data through its PostgreSQL schema provider. The UI exports predictions and trips as CSV/JSON without persisting export artifacts.

## Retention And Deletion

No automated retention policy is currently implemented. Future public deployment should define retention for reports, photos, and trip plans.

## Privacy Notes

Reports and trips can contain user-authored descriptions and photo URLs. Treat public deployment as requiring validation and moderation controls.

## Fallback Data

The API intentionally returns sample regions and foliage data when PostgreSQL is unavailable for discovery flows. This keeps the map and CLI inspectable in degraded development states while write operations still require the database. The behavior is implemented in [CODE: api/main.go#sampleRegionsDataset] and [CODE: api/main.go#foliageHandler].

## Ownership

PostgreSQL is the source of truth for persisted user content and generated predictions. The UI may export current in-memory region and trip state to CSV/JSON, but exported files do not become canonical data.

## Requirement Coverage

- Region and foliage reads: [REQ: REQ-P0-003]
- Weather reads: [REQ: REQ-P0-004]
- User reports and photos: [REQ: REQ-P1-001], [REQ: REQ-P1-004]
- Trip plans: [REQ: REQ-P1-003]
- Prediction storage: [REQ: REQ-P0-005], [REQ: REQ-P2-001]

## Cross-References

- [DOC: docs/concepts/ARCHITECTURE.md]
- [DOC: docs/concepts/DOMAINS.md]
- [DOC: docs/reference/api-endpoints.md]
