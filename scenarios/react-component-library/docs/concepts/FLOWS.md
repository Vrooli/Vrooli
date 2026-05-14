# Flows — React Component Library

This document is the canonical workflow map for ordered behavior.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Validation |
|---|---|---|---|---|
| Index library | components | CLI/API/UI `components index` | Manifests and version folders are validated and reflected in SQLite | Indexer, repository, handler, CLI tests |
| Apply component | adoptions | CLI/API/UI `adoptions apply` | Selected version source is copied into a target scenario with provenance and an adoption row | Service, handler, CLI, UI tests |
| Refresh drift | adoptions | CLI/API/UI `adoptions refresh` | Adoption rows receive separate library-version and local-edit statuses | Service status matrix and UI tests |
| Reapply component | adoptions | CLI/API `adoptions reapply` | Adopted file is overwritten from a selected version; local edits require confirmation | Service and handler tests |
| Diff versions/adoptions | versions | CLI/API/UI diff request | Server returns aligned line diff rows | Versions service and handler tests |

## Apply Component

1. Resolve component by id.
2. Use requested version, or the manifest latest when no version is
   supplied.
3. Reject an existing target path unless overwrite is confirmed.
4. Write a provenance header with library id, version, adoption id,
   applied timestamp, and source sha.
5. Copy the full editable source body into the target scenario.
6. Insert the adoption row with source and adopted snapshot hashes.

## Refresh Drift

Refresh computes two dimensions:

- `library_version_status`: `current`, `behind`, `deprecated`,
  `missing`, or `unknown`.
- `local_status`: `clean`, `modified`, `missing`, or `unknown`.

This lets the UI distinguish a clean but behind copy from a locally
edited copy that is also behind.

## Deferred Flows

| Flow | Risk | Next Step |
|---|---|---|
| Draft/release lifecycle UI | Released version immutability is enforced by convention, not a full workflow model. | Add draft/release commands and tests when editing releases expands. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`DATA.md`](DATA.md)
- [`../internal/TESTING.md`](../internal/TESTING.md)
