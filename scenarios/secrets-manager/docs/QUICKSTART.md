# Quickstart — Secrets Manager

## Prerequisites

- A Vrooli checkout with lifecycle tooling available.
- The credential authority available through the host key service or encrypted authority storage.
- Postgres configured for the scenario metadata.
- Node and Go toolchains supplied by the Vrooli environment.

## 1 — Setup

Run `make setup` from `scenarios/secrets-manager`.

## 2 — Start

Run `make start`. The lifecycle assigns `API_PORT` and `UI_PORT`.

## 3 — Open

Run `make open`, or inspect assigned ports with `make status`.

## 4 — Inspect posture

Run `secrets-manager status` for health, credential-authority coverage, and compliance posture.

## 5 — Run the tests

Run `vrooli scenario test secrets-manager`. Test Genie owns the run; use the recorded run evidence rather than launching application processes directly.

## Troubleshooting

See [Troubleshooting](guides/troubleshooting.md) and [Runbook](operations/RUNBOOK.md).
