# Domains

## Purpose Of This Document

This document maps the scenario's product domains to implementation surfaces.

## Domain Inventory

| Domain | Purpose | Source Paths |
| --- | --- | --- |
| Regions | Read-only mapped foliage areas. | [CODE: api/main.go#regionsHandler], [CODE: cli/domains/regions/register.go] |
| Foliage | Status, weather, and peak prediction. | [CODE: api/main.go#foliageHandler], [CODE: cli/domains/foliage/register.go] |
| Reports and photos | Crowd reports and gallery photo URLs. | [CODE: api/main.go#reportsHandler], [CODE: cli/domains/reports/register.go] |
| Trips | Multi-region itinerary storage. | [CODE: api/main.go#tripsHandler], [CODE: cli/domains/trips/register.go] |
| Exports | Client-side CSV/JSON downloads. | [CODE: ui/src/app.js] |

## Domain Details

## Regions

Regions are read-only map entities surfaced by `GET /api/regions`, the UI Regions tab, and the root CLI list command. Implementation references: [CODE: api/main.go#regionsHandler], [CODE: cli/domains/regions/register.go], [CODE: ui/src/app.js].

## Foliage

Foliage covers current status, weather lookup, and peak prediction. It maps to `GET /api/foliage`, `GET /api/weather`, and `POST /api/predict`. Implementation references: [CODE: api/main.go#foliageHandler], [CODE: api/main.go#weatherHandler], [CODE: api/main.go#predictHandler], [CODE: cli/domains/foliage/register.go].

## Reports And Photos

Reports and gallery photos share the `user_reports` table and `GET/POST /api/reports` endpoint. A report may include `photo_url`, which the UI gallery renders and filters. Implementation references: [CODE: api/main.go#reportsHandler], [CODE: cli/domains/reports/register.go], [CODE: ui/src/app.js].

## Trips

Trip plans are saved multi-region itineraries backed by `trip_plans` and exposed through `GET/POST /api/trips`. Implementation references: [CODE: api/main.go#tripsHandler], [CODE: cli/domains/trips/register.go].

## Exports

Exports are client-side CSV/JSON downloads generated from loaded region and trip data. They support [REQ: REQ-P2-003] without adding server-side file storage.

## Shared Concepts

All domains use the API response envelope documented in [DOC: docs/reference/api-endpoints.md]. Region IDs are the cross-domain join key.

## Deferred Domains

Real-time weather ingestion and Redis-backed caching are declared in configuration but not implemented as first-class domains yet.

## Non-Domains

Authentication, social networking, and satellite imagery processing are non-goals in the PRD.

## Cross-References

- [DOC: docs/concepts/ARCHITECTURE.md]
- [DOC: docs/concepts/FLOWS.md]
- [DOC: docs/concepts/DATA.md]
- [DOC: docs/internal/SEAMS.md]
