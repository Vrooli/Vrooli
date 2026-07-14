# Integrations

## Purpose Of This Document

This document records resource and service dependencies, their failure modes, and ownership.

## Dependency Inventory

- PostgreSQL
- Redis
- Ollama
- Qdrant
- browser-automation-studio optional fallback
- App Monitor iframe bridge

## Vrooli Resources

### PostgreSQL

PostgreSQL stores regions, observations, weather data, predictions, reports, and trips. The API connects through the shared Vrooli database helper in [CODE: api/main.go#initDB]. Schema ownership lives in [CODE: initialization/postgres/schema.sql].

### Redis

Redis is declared as a required resource in [CODE: .vrooli/service.json] for real-time weather caching. The current API code does not yet use Redis directly; treat this as an operational dependency reserved for caching improvements.

### Ollama

Ollama powers advanced peak prediction through direct API calls. The API reads `OLLAMA_URL`, posts to `/api/generate`, validates the JSON response, and falls back when unavailable. Implementation: [CODE: api/main.go#generateFoliagePrediction].

### Qdrant

Qdrant is declared for generated embeddings and AI assistance during setup.

## Scenario Dependencies

This scenario has no scenario-to-scenario runtime dependencies.

## Third-Party Services

No external third-party production service is required. Weather data is currently modeled as PostgreSQL records.

## App Monitor Iframe Bridge

The UI initializes the iframe bridge when embedded and resolves proxy metadata before falling back to loopback. Implementation: [CODE: ui/src/app.js#resolveApiBase].

## Weather

Weather data is read from PostgreSQL via `GET /api/weather`. The PRD now describes direct API/CLI flows rather than n8n workflows for weather data collection.

## Failure Modes

PostgreSQL failures degrade read-only discovery but block writes. Ollama failures should fall back to typical peak-week prediction. UI proxy metadata failures fall back to loopback and can cause smoke failures when the API is not running.

## Cross-References

- [DOC: docs/reference/configuration.md]
- [DOC: docs/operations/RUNBOOK.md]
