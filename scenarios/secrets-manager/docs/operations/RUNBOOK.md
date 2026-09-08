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

Data Backup Manager owns target registration, encrypted snapshot execution,
retention, restore, and recovery-drill evidence. Secrets Manager registers its
owned targets through the typed safety contract and projects the provider's
coverage at `/api/v1/recovery/status`; a verified backup alone does not make
the vault recovery-ready until a restore drill is verified. External provider
values are reported as dependencies or references when they are not inside the
encrypted snapshot. A recorded credential recovery bundle is an import
artifact and is rejected as an ordinary backup target.

### Replacement-host recovery

Use two independently stored artifacts: the Data Backup Manager restore
receipt/snapshot and the credential-authority recovery bundle with its separate
passphrase or key material. Inspect the encrypted bundle first; the preview
must report only identities and fields. Restore into a disposable authority
and verify synthetic values, item inventory, relationships, and source
bindings before considering activation. A wrong passphrase, missing key,
corrupt artifact, or unavailable provider is an incomplete recovery and must
not produce restored values or replace the active authority.

After isolated usability succeeds, request fresh human assurance for
`recovery:activate` and POST `/api/v1/recovery/activate` with the expected
epoch and one-time assurance token. Activation advances the per-workspace
recovery epoch, invalidates broker/browser/SSH capabilities and pending
assurance/export handles, and moves restored grants to `recovery_review`.
Reconcile current membership and provider revocation, then issue new grants.
An old capability must remain denied even if a restored metadata row still
labels it active. If activation is interrupted, discard the isolated target;
the original active authority remains the source of truth.

The recovery kit checklist is: independent encrypted bundle, independent
passphrase or key, source and inventory manifest, DBM restore receipt,
authority usability receipt, current membership/provider reconciliation, and
the recovery activation audit event. Never store the passphrase, plaintext
values, or restored contents in DBM drill records, logs, UI state, or audit
details.

## Maintenance Tasks

Run `vrooli scenario test secrets-manager`, inspect `secrets-manager status`, and review deployment readiness before a release.

## Escalation

Escalate missing signing authority, host privilege, or broker authorization to the operator or CI owner. Include sanitized lifecycle and Test Genie evidence.

## Cross-References

- [Deployment](DEPLOYMENT.md)
- [Troubleshooting](../guides/troubleshooting.md)
