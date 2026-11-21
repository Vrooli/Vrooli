# artifactledger

Scenario-facing door to the control plane's artifact removal ledger.

Scenarios that remove install-root artifacts (storage-manager's orphan reaper is the main
one) need the same receipts as control-plane removals. Scenario modules cannot import
`internal/artifactledger`, so this package aliases its types in the same shape as
`packages/hostreq`.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
