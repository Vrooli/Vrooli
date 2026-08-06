# Secrets Manager portability audit

## Scope

This audit covers the API path used by a Tier 2 desktop bundle. It does not
claim a signed release, macOS validation, or Windows validation.

## Implemented portability contract

The desktop runtime sets `VROOLI_DESKTOP_MODE=true` and supplies an absolute
`APP_DATA_DIR`. The API then uses the `api-core/storage` desktop resolver with
the variant-aware `secrets-manager` namespace. The resolved SQLite metadata
database is private to the desktop app-data root.

The SQLite driver is pure Go. `CGO_ENABLED=0 go build ./...` succeeds for the
API. The approved dependency record limits `modernc.org/sqlite` v1.54.0 to the
Secrets Manager desktop-private-storage use case.

## Compatibility and migration

Earlier bundles stored metadata at `APP_DATA_DIR/runtime/api/secrets-manager.sqlite`.
At startup, the API checkpoints that SQLite database and moves it to the
resolver-owned data path when the new database does not yet exist. This keeps
existing private metadata during the storage-layout upgrade.

The migration and private metadata query are covered by
`TestDesktopDatabaseMigratesLegacyPrivateState` and
`TestDesktopDatabaseProvidesPrivatePortableSecretMetadata`.

## Current evidence

| Target | Evidence | Status |
| --- | --- | --- |
| Linux API build | `CGO_ENABLED=0 go build ./...` | Pass |
| Desktop private database | Focused desktop storage tests | Pass |
| Linux desktop pipeline | A fresh authority-backed release stage was built and every `SHA256SUMS` entry was verified. The pipeline correctly rejected it because the external release signer has not supplied `SHA256SUMS.sig`. | Fail closed; release-signing handoff required |
| Linux native authority | Starting Secrets Manager correctly rejects an unmet Linux Secret Service `secret-tool` host requirement before credential operations. | Host installation required |
| Headless host with no native credential store | With `secret-tool` removed from `PATH`, `vrooli credentials store init` created the encrypted file store and a full provision → status → delete cycle ran through the CLI. A resource declaring a required credential reported `Healthy: yes, Status: configured`, and reported unhealthy again after the credential was deleted. The provisioned value appeared in no command output, log, or file. | Pass |
| macOS and Windows | No signed artifact or target-host test evidence in this run | Not claimed |

## Remaining release prerequisites

The Tier 2 acceptance gate requires the release signing authority to sign the
fresh `SHA256SUMS` manifest, followed by a successful smoke test against that
exact signed artifact. The source build deliberately does not create
`SHA256SUMS.sig`, and no local substitute is acceptable.

The Tier 1 acceptance gate requires the Linux Secret Service client. Install
the declared host tool through the control plane, then start Secrets Manager
through its lifecycle and rerun the server-owned Test Genie suite. The missing
tool is intentionally a hard failure; the API does not fall back to plaintext
storage.
