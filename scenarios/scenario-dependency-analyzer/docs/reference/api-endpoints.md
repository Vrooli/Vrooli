# API Endpoints

## System

The API exposes health, scenario catalog, declared graph, actual interface graph, drift, deployment readiness, and generated Connect RPC surfaces. Legacy REST details remain in `../api.md`.

## Notes (CRUD reference)

SDA is analysis-oriented rather than CRUD-heavy. Mutating endpoints are limited to scan/apply and generated bundle or deployment artifacts.

## Adding a new endpoint

Add the handler, tests, generated proto/Connect contract when the surface is programmatic, `.vrooli/endpoints.json` metadata, and this reference entry.

## Cross-references

- `../../.vrooli/endpoints.json`
- `../api.md`
- `configuration.md`
