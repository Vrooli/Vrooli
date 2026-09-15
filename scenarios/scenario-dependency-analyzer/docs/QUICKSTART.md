# Quickstart

## Prerequisites

Use the Vrooli lifecycle stack with `proto-health`, `code-facts`, SQLite, Ollama, and Qdrant available through scenario/resource dependencies.

## 1 — Setup

Run `vrooli scenario setup scenario-dependency-analyzer` from the repository root when binaries or UI dependencies are stale.

## 2 — Start

Run `vrooli scenario start scenario-dependency-analyzer` or `make start` from this scenario directory.

## 3 — Open

Use lifecycle-reported ports. The UI serves the graph, catalog, and deployment readiness views.

## 5 — Run the tests

Run `vrooli scenario test scenario-dependency-analyzer` for the full scenario suite. Use focused Go and UI commands only while iterating.

## Cross-References

- `operations/RUNBOOK.md`
- `reference/configuration.md`
