# Deployment

## Purpose Of This Document

Capture supported deployment tiers and release assumptions.

## Supported Tiers

SDA currently targets local Vrooli lifecycle deployment and future desktop/server bundles.

## Runtime Requirements

Requires lifecycle-managed API/UI processes, SQLite path configuration, and healthy upstream fact scenarios for actual graph completeness.

## Packaging

Bundle metadata is generated through the deployment domain and should include runtime assets, SQLite path conventions, and health checks.

## Release Checklist

Run focused API/CLI/UI tests, `packages/proto` lint after proto changes, and `vrooli scenario test scenario-dependency-analyzer`.

## Rollback

Rollback by reverting code/config changes and regenerating proto artifacts when generated contracts changed.

## Cross-References

- `RUNBOOK.md`
- `../reference/configuration.md`
