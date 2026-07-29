# Data

## Purpose Of This Document

Describe the data that Vrooli Onboarding owns and the data it deliberately
projects without owning.

## Storage Overview

The sole durable onboarding record is `.vrooli/operator-state.json`, written
atomically by the API. Credential values remain in the native credential
authority and are never written to operator state, browser storage, or the
scenario database.

## Data Ownership

Onboarding owns scenario selections, optional-resource choices, operating
mode, and the current operator intent. Resource manifests own descriptors;
credential authorities own values.

## Schema Map

`operator-state.schema.json` defines the persisted record. V2 read models are
derived views and have no independent schema or storage.

## Migrations And Compatibility

V2 replaces the former progress record. Legacy progress is not a fallback or
configuration authority.

## Import / Export

Operator state is inspectable configuration. Credential recovery is performed
only through the encrypted control-plane recovery export and restore commands.

## Retention And Deletion

Deleting operator state resets onboarding choices. Credential deletion is an
explicit authority operation and is not implied by reset.

## Privacy Notes

No credential values, tokens, or recovery passphrases are stored in this
scenario's files, logs, API responses, or UI state.

## Cross-References

- [Domains](DOMAINS.md)
- [Wizard flow](../WIZARD_FLOW.md)
