# Integrations

## Test Genie

Test Genie uses Tidiness Manager for the `tidiness` phase. It uses Quality Health for the `quality` phase. This split is the current ownership contract.

## Quality Health

Quality Health validates Tidiness Manager's own lint/type/static-quality setup. It also owns the static-quality policy that used to be mixed into Tidiness Manager and Scenario Auditor.

## visited-tracker

visited-tracker can provide visit history for smart prioritization and campaign handoffs. Tidiness Manager wraps some visited-tracker workflows through the `tracking` CLI domain.

## resource-claude-code And resource-codes

Smart scans can use AI resources to produce deeper refactor recommendations. The light and tidiness scan paths must continue to work without those resources.

## PostgreSQL

The API uses PostgreSQL for issues, metrics, campaigns, and history. Database configuration comes from lifecycle-managed environment variables.

## Shared Source Evidence

Code Facts owns source and graph evidence. API Core's `factgraph` package indexes those graphs once by target-relative path and language, including import edges, so Tidiness file metrics, detailed metrics and duplicate-body detectors share one normalized projection. The same body-group detector feeds the single `tidiness-manager/internal/discovery` registry and ranked recommendation endpoint. Go, TypeScript, and JavaScript detectors are calibrated separately. The Go `dupl` subprocess and output parser have been removed.

`typescript-code-graph` remains the TypeScript and JavaScript parser owner, reached through Code Facts, and emits language-tagged normalized function-body identities for Tidiness. Go, TypeScript and JavaScript detectors have separate calibration. JavaScript now has a complete prompt-manager source census. Its target binding uses the authoritative Code Facts `source_hash` and `config_hash` from a fast `parse_units` response; the JS-only manifest hash is retained separately and is not presented as a Code Facts fingerprint. The local registry owner-path replay measured two exact groups against the complete singleton pool. The full graph recommendation request timed out, so served graph results remain unverified. The rates only apply to prompt-manager while those Code Facts hashes match; they are not a fleet estimate. TypeScript still has synthetic positive/negative regressions but no source-replayed census. The old Go `dupl` branch and all `jscpd` paths are removed. Scenario Auditor remains the standards-rule runner, Architecture Cartographer owns architecture policy and audit snapshots, Flow Verifier owns lifecycle inference, and Template Manager owns template provenance. These are distinct policy and workflow responsibilities; the shared facts owner does not replace them.

If Code Facts or a required provider is unavailable, Go source metrics report degraded metadata instead of claiming that no duplicates exist.

The ranked recommendation endpoint also reads Scenario Auditor's cached
`/api/v1/standards/violations` results for the requested scenario. The unified
registry exposes these as standards findings, retaining rule ID, severity,
standard, source, file, and remediation guidance. They remain uncalibrated and
carry no expected line reduction because a policy violation is not a reduction
prediction. Auditor unavailability adds a warning while leaving Code Facts
recommendations available; findings outside the requested scenario root are
discarded with a warning. Scenario Auditor continues to own rule definitions,
policy, and standards scans.
