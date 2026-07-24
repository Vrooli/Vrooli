# Security — Secrets Manager

## Purpose Of This Document

This document defines the scenario's security boundaries.

## Data Sensitivity

Secret values, Vault tokens, private keys, and credential files are never returned by the API, stored in metadata tables, or written to test artifacts.

## Auth And Authorization

The scenario’s user-facing API is lifecycle-local. Vault scoped use is brokered. A use lease cannot obtain management authority.

## Secrets

Vault resource configuration declares secret storage and Linux Secret Service tooling. Artifact provenance verifies upstream identity and release checksum signatures before bundle admission.

## Threat Model

Primary risks are secret-value disclosure, direct remote Vault bypass, ambient shared-resource use by desktop bundles, stale deployment strategy, and unsigned artifacts.

## Security Gaps

Live Linux Secret Service validation and external release signing remain operator/CI-owned gates. See `PROBLEMS.md` for tracked work.

## Cross-References

- [Integrations](../concepts/INTEGRATIONS.md)
- [Deployment](../operations/DEPLOYMENT.md)
