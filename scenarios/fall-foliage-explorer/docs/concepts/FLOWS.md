# Flows

## Purpose Of This Document

This document records user and system flows, including lifecycle states and validation evidence.

## Flow Inventory

- Foliage discovery
- Prediction
- Crowd report to gallery
- Trip planning and export

## Flow Details

### Foliage Discovery

Users load the UI, the app resolves the API base, retrieves regions, and renders foliage status overlays on the map. This flow is implemented in [CODE: ui/src/app.js] and backed by [CODE: api/main.go#regionsHandler] plus [CODE: api/main.go#foliageHandler].

Validated by [REQ: REQ-P0-003], [REQ: REQ-P0-006], and tests in [CODE: api/main_test.go].

### Prediction

Users or CLI operators request a peak prediction for a region. The API loads region metadata, calls Ollama when available, stores successful predictions, and falls back to typical peak-week logic when the model path fails. Implementation: [CODE: api/main.go#predictHandler] and [CODE: api/main.go#generateFoliagePrediction].

Validated by [REQ: REQ-P0-005], [REQ: REQ-P2-001], and [CODE: api/business_test.go#TestPredictionWorkflow].

### Crowd Report To Gallery

Users submit a foliage report with optional photo URL, then browse reports by region or date in the gallery. The API stores the submitted report in PostgreSQL. Implementation: [CODE: api/main.go#reportsHandler] and [CODE: ui/src/app.js].

Validated by [REQ: REQ-P1-001], [REQ: REQ-P1-004], and [CODE: api/integration_test.go#TestSubmitReportIntegration].

### Trip Planning And Export

Users save trip plans, retrieve saved plans, then export trips as CSV or JSON from the UI. Implementation: [CODE: api/main.go#tripsHandler] and [CODE: ui/src/app.js].

Validated by [REQ: REQ-P1-003], [REQ: REQ-P2-003], and [CODE: api/integration_test.go#TestTripPlanningIntegration].

## State Machines

The current implementation does not define a formal state machine. UI tab state, form state, and export state are local browser state.

## Maturity Ladder

Core flows are implemented and covered by API/integration tests. BAS playbook registry coverage is still missing.

## Production Shape

Production operation should keep lifecycle-owned API and UI processes healthy, ensure PostgreSQL schema bootstrap is idempotent, and start Ollama before prediction-heavy validation.

## Deferred / Unmodeled Flows

Weather ingestion scheduling, Redis cache invalidation, moderation, and authentication are deferred.

## Cross-References

- [DOC: docs/concepts/ARCHITECTURE.md]
- [DOC: docs/concepts/DOMAINS.md]
- [DOC: docs/concepts/DATA.md]
- [DOC: docs/operations/RUNBOOK.md]
