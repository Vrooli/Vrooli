# Observability — Secrets Manager

## Purpose Of This Document

This document describes operator-visible signals without recording secret values.

## Signals

Health, credential-authority coverage, compliance, deployment readiness, scan
status, and campaign state are the primary signals.

## Logs

Use `make logs` or `vrooli scenario logs secrets-manager`. Logs may contain resource and scenario identifiers but must not include secret values or access tokens. The API logger applies a final key/value and bearer-token redaction guard; audit details use the same guard and remain metadata-only.

## Metrics

The scenario reports metadata-derived posture such as required-secret coverage and vulnerability severity distribution. It does not treat secret values as telemetry.

Credential-use activity records successful and failed destination operations. Each
durable event has a workspace-scoped idempotency key and integrity digest, so a
retry after restart does not create a second logical event. `/api/v1/audit`
reports `verified`, `unknown`, or `tampered` evidence status, while
`/api/v1/audit/export` emits the same metadata-only projection with
`payload_policy: metadata_only`.

The iframe bridge captures no network bodies. Secrets Manager disables network
capture for its product iframe, and bridge metadata redacts credential-shaped
fields and sensitive query parameters before host delivery.

## Alerts / Health

Use `/health` and `/api/v1/health` for lifecycle-facing readiness. A missing required resource is an actionable degraded condition.

## Telemetry Gaps

Scheduled deployment-metadata freshness and historical trend alerts remain deferred; see `../internal/PROBLEMS.md`.

## Cross-References

- [Runbook](RUNBOOK.md)
- [Security](../internal/SECURITY.md)
