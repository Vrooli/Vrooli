# Observability — Secrets Manager

## Purpose Of This Document

This document describes operator-visible signals without recording secret values.

## Signals

Health, Vault coverage, compliance, deployment readiness, scan status, and campaign state are the primary signals.

## Logs

Use `make logs` or `vrooli scenario logs secrets-manager`. Logs may contain resource and scenario identifiers but must not include secret values or access tokens.

## Metrics

The scenario reports metadata-derived posture such as required-secret coverage and vulnerability severity distribution. It does not treat secret values as telemetry.

## Alerts / Health

Use `/health` and `/api/v1/health` for lifecycle-facing readiness. A missing required resource is an actionable degraded condition.

## Telemetry Gaps

Scheduled deployment-metadata freshness and historical trend alerts remain deferred; see `../internal/PROBLEMS.md`.

## Cross-References

- [Runbook](RUNBOOK.md)
- [Security](../internal/SECURITY.md)
