# Security — Secrets Manager

## Purpose Of This Document

This document defines the scenario's security boundaries.

## Data Sensitivity

Secret values, authority management credentials, private keys, and credential files are never returned by the API, stored in metadata tables, or written to test artifacts.

## Auth And Authorization

The scenario’s user-facing API is lifecycle-local. Credential-authority use is brokered through the control plane; a use lease cannot obtain management authority.

## Secrets

Credential descriptors declare authority identities and Linux Secret Service tooling. Artifact provenance verifies upstream identity and release checksum signatures before bundle admission.

## Threat Model

Primary risks are secret-value disclosure, direct authority bypass, ambient shared-resource use by desktop bundles, stale deployment strategy, and unsigned artifacts.

## Security Gaps

Live Linux Secret Service validation and external release signing remain operator/CI-owned gates. See `PROBLEMS.md` for tracked work.

### Dependency Remediation Status (2026-07-29)

Governed upgrades moved the UI to ESLint 10, TypeScript-ESLint 8.65, Vitest 2,
Vite 6, Tailwind 4, and the maintained `eslint-plugin-import-x` successor. A
governed pnpm resolver override selects the compatible `minimatch` 10 line for
that successor. The Electron package lock and every remaining UI dependency path
now select `brace-expansion` 5.0.8. Security Health passes with zero error-level
findings; UI lint and all 96 UI tests pass on the upgraded graph.

## Cross-References

- [Integrations](../concepts/INTEGRATIONS.md)
- [Deployment](../operations/DEPLOYMENT.md)
