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

### Dependency Remediation Status (2026-07-29)

Governed resolver pins removed the remediable `picomatch`, `minimatch`,
`flatted`, and Electron packaging findings. The Electron package lock now selects
`brace-expansion` 5.0.8. UI lint and all 96 UI tests pass with the updated graph.

One Security Health error remains: `GHSA-mh99-v99m-4gvg` applies to every
`brace-expansion` release through 5.0.7, including the 1.x and 2.x copies required
by the supported ESLint 9 / TypeScript-ESLint 8 development-tool graph. The only
published patched line is 5.0.8, whose module API is not compatible with those
older consumers. A governed ESLint 10 upgrade was evaluated to remove the legacy
consumer path, but TypeScript-ESLint 8.47.0 fails to initialize with ESLint 10.
Do not hand-edit lockfiles, force an incompatible cross-major override, or suppress
this finding; resolve it with a coordinated upstream-compatible lint-tool upgrade.

## Cross-References

- [Integrations](../concepts/INTEGRATIONS.md)
- [Deployment](../operations/DEPLOYMENT.md)
