# Deployment — Secrets Manager

## Purpose Of This Document

This document defines the safe deployment posture for Secrets Manager.

## Supported Tiers

Tier 1 uses Vrooli-managed shared Vault through a scoped broker lease. Tier 2 desktop bundles use private Vault by default. Shared reuse requires explicit consent and an authorized broker binding.

## Runtime Requirements

Vault and Postgres are required scenario resources. Linux shared Vault requires the declared Secret Service host-tool capability. Artifact staging requires verified provenance and a release signature for bundle admission.

## Packaging

Scenario-to-desktop packages the scenario and a Vault server artifact. The bundle must include its artifact inventory and must not depend on an ambient development Vault.

## Release Checklist

1. Verify the Vault artifact and detached checksum signature.
2. Validate resource selection for the target tier.
3. Start the bundle with private Vault.
4. Exercise the secret workflow without exposing values.
5. Exercise explicit shared reuse and denied-grant fallback.

## Rollback

Stop the lifecycle-managed bundle. Preserve metadata and evidence. Do not copy Vault data between roots without an explicit migration decision.

## Cross-References

- [Integrations](../concepts/INTEGRATIONS.md)
- [Runbook](RUNBOOK.md)
