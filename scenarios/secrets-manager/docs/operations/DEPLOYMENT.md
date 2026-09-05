# Deployment — Secrets Manager

## Purpose Of This Document

This document defines the safe deployment posture for Secrets Manager.

## Supported Tiers

Tier 1 and Tier 2 use the canonical credential authority. Desktop bundles use
the host key service or encrypted authority storage and operator-controlled
recovery bundles. Vault is packaged only for an explicitly governed
Vault-specific capability.

## Runtime Requirements

Postgres is the only required scenario resource. Credential operations require a
supported native key service or encrypted authority store. Artifact staging
requires verified provenance and a release signature for bundle admission.

## Packaging

Scenario-to-desktop packages the scenario and its authority configuration. The
bundle must include its recovery policy and must not depend on an ambient
development Vault.

## Release Checklist

1. Verify the authority/runtime artifact inventory and detached checksum signature.
2. Validate resource selection for the target tier.
3. Start the bundle with its native/encrypted credential authority.
4. Exercise the secret workflow without exposing values.
5. Exercise recovery-bundle validation without printing credential values.

## Rollback

Stop the lifecycle-managed bundle. Preserve metadata and evidence. Keep the
recovery bundle and passphrase separate; do not copy authority data between
hosts without an explicit migration decision.

## Cross-References

- [Integrations](../concepts/INTEGRATIONS.md)
- [Runbook](RUNBOOK.md)
