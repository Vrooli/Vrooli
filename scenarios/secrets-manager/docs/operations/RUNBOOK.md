# Runbook — Secrets Manager

## Purpose Of This Document

This document gives the supported operational entrypoints.

## Start / Stop / Status

Run `make setup`, `make start`, `make status`, `make logs`, and `make stop` from the scenario directory. These commands delegate to the Vrooli lifecycle.

## Common Incidents

- Missing native key-service or encrypted-authority support: follow the credential-authority doctor remediation and use a recovery bundle when moving hosts.
- Database unavailable: check Postgres lifecycle health and confirm whether desktop mode is expected.
- Bundle artifact rejection: obtain a correctly signed release checksum manifest; do not bypass provenance verification.

## Backup / Restore

Credential values and metadata are distinct stores. Back up and restore only
through their owning supported mechanisms. Keep recovery bundles and
passphrases separate and never copy plaintext values into scenario metadata.

## Maintenance Tasks

Run `vrooli scenario test secrets-manager`, inspect `secrets-manager status`, and review deployment readiness before a release.

## Escalation

Escalate missing signing authority, host privilege, or broker authorization to the operator or CI owner. Include sanitized lifecycle and Test Genie evidence.

## Cross-References

- [Deployment](DEPLOYMENT.md)
- [Troubleshooting](../guides/troubleshooting.md)
