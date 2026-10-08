# Documentation Progress

Recovery locations below are relative to the control-plane runtime home
(`~/.vrooli` for the invoking user); entry names follow
[the runtime-home contract](../reference/storage-retention.md#control-plane-runtime-home-contract).
Archives remain local to the originating operator's installation. Copy protected
backups and plan artifacts with that installation's durable backups.

## Documentation capture preservation — 2026-09-05

Retired redundant integrations-plan fragments and dated captures after comparison
with their owners. Unique sources remain in [the plan source index](../plans/README.md).
Recovery: `backups/docs-cleanup-20260905T040747Z/` contains
`originals.tar.gz` (222 original files), `manifest.json`, and
`cleanup-receipt.json`. Archive SHA-256:
`f7dfeeec124d5c90894a97ad337b2361602df3e9ae75f8de94c8fffd3575b250`.

## Documentation cleanup — 2026-09-07

Relocated plan supplements and detailed progress history with verified copies.
Historical sources remain beneath the protected `plan_artifacts` entry:

| Material | Path beneath `plan-artifacts/` |
|---|---|
| Prompt Manager world sources | `living-world-20260905/` |
| Portal Everywhere sources and evidence | `portal-everywhere-20260905/` |
| Cross-platform agreements and captures | `docs-history-20260907/docs/reference/cross-platform-effort/` |
| Internal-debt audits and declared-scenario report | `docs-history-20260907/docs/architecture/` |
| Condensed scenario progress logs | `docs-history-20260907/scenarios/<scenario>/docs/internal/PROGRESS.md` |

Exact originals, SHA-256 manifests, dispositions, and plan reference-update
receipts: `backups/docs-cleanup-20260907-snpJq5/`.
Use each archive's manifest to locate an original, restore it to a separate
destination, and verify its hash. Relocation does not change plan acceptance.

## Legacy audit and research retirement — 2026-09-07

Archived 22 tracked audit/research snapshots with no scenario-local inbound
references and no concurrent working-tree edits. They were historical inputs,
not current contracts; files with active references remain in place for a later
canonical migration into architecture, seams, problems, decisions, progress,
or reference docs.

Recovery: `plan-artifacts/docs-cleanup-20260907-legacy-audits/` with
`manifest.sha256`. The exact repository-side Portal working corpus is also
preserved at `plan-artifacts/portal-everywhere-20260905-repository-snapshot-20260907/`;
the active canonical Portal artifacts remain at `plan-artifacts/portal-everywhere-20260905/`.

## Current documentation and historical owners

Current contracts: [package families](../../internal/README.md),
[build and validation](../reference/build-and-validation.md),
[platform support](../reference/platform-support.md),
[declared scenario](../architecture/the-declared-scenario.md), and
[System Monitor machine selection](../../scenarios/system-monitor/docs/concepts/ARCHITECTURE.md#machine-selection-and-remote-presence).

The 2026-09-06 internal-consolidation handoff recorded incomplete before-state
receipts, missing CLI goldens, and unrelated validation debt. Its
[preserved evidence](#documentation-cleanup-completion--2026-09-07)
is retained. Read current status with
`plan-manager plans get consolidate-internal-delete-the-dead-surface-close-the`.
The full historical plan roster and dated subsystem results are in the
[tracker archive](#tracker-condensation-and-portable-references--2026-09-07).

## Documentation cleanup follow-up — 2026-09-07

Consolidated shared UI guidance in Template Manager; preserved Backdrop Studio
captures while keeping its regression corpus in integration testdata. Retired
legacy plans, nine unreferenced HTML snapshots, web-search execution artifacts,
and the Git Control Tower placeholder audit after preservation.

| Recovery purpose | Runtime-home location |
|---|---|
| Originals, hashes, dispositions | `backups/docs-cleanup-20260907-followup/` |
| Readable historical sources | `plan-artifacts/docs-cleanup-20260907-followup/` |
| Import/reference receipts and owner mappings | Same artifact root: `owner-receipts/`, `imported-plans.json`, `plan-relocations.json` |
| HTML, web-search output, placeholder audit | Same artifact root: `current-doc-audit/` |

### Historical report sources

Fleet-supervision, tools/safeguards, and speech-capability readouts dated
2026-08-28 through 2026-08-30 remain under
`plan-artifacts/docs-cleanup-20260907-followup/docs/reports/`.
Current semantics belong to the [capability registry](../../packages/capability-registry-go/README.md)
and [resource architecture](../resources/architecture.md#control-plane-host-ownership).

## Tracker condensation and portable references — 2026-09-07

Condensed the root trackers and replaced operator-specific source paths with
relative links and portable evidence locators. Full pre-edit tracker text is
preserved under `backups/docs-trackers-portability-20260907-I5CiLV/docs/internal/`:

| Original | SHA-256 |
|---|---|
| `PROGRESS.md` | `d7699c7a45685846629489bfdfad0351bbec7f0c6413a2ea52e687590c57731d` |
| `PROBLEMS.md` | `5067e7754ae1d286a86a6fc878c743aaf64c59bdcf3bb480ba0c132f4e7ee9ca` |

This preserves detailed investigations, measurements, recovery instructions,
and unresolved subsystem handoffs. [Problems](PROBLEMS.md) indexes follow-up
owners; historical observations require rechecking before further work.

## Documentation cleanup completion — 2026-09-07

Reconciled the Stream of Consciousness Analyzer issue ledgers, corrected host-tool
and documentation-maintenance commands, condensed large scenario progress logs,
and relocated dated reviews, test summaries, and plan execution captures.
Unresolved handoffs and missing historical evidence remain explicit in owner docs.

Recovery locations are relative to the protected control-plane runtime home:

| Material | Recovery location |
|---|---|
| Exact originals, hashes, dispositions and owner-update receipts | `backups/docs-cleanup-20260907-final-txumdx73/` |
| Readable historical sources, retaining original repository-relative paths | `plan-artifacts/docs-cleanup-20260907-final-txumdx73/` |

Use `manifest.json` in the backup directory to locate each original under
`originals/` and verify its SHA-256 before restoring to a separate destination.
Copy these protected local archives with the originating installation's durable
backups. The source tree carries recovery locators; it does not replicate the
archived bytes. Relative links inside an untouched historical source describe
its original repository layout.

Plan-owned sources retain their original relative path beneath the artifact root:

- Internal consolidation: `docs/architecture/evidence/internal-consolidation/`;
  owner `consolidate-internal-delete-the-dead-surface-close-the`. Incomplete
  before-state receipts and absent CLI goldens are not resolved by relocation.
- First-run handoff: `scenarios/vrooli-onboarding/docs/evidence/first-run-handoff/`;
  owner `first-run-handoff-completion-truth-and-structural-cleanup`.
- Credential review: `docs/configuration/secrets-architecture-review.html`;
  owners `consolidate-the-credential-seam-and-harden-the-desktop` and
  `credential-stack-hardening-fleet-distribution-and-keyring`.
- Audio review: `docs/design/audio-reliability-audit-2026-08-09.html`;
  owners `trustworthy-long-form-dictation-make-the-runtime-knowable` and
  `audio-dictation-reliability-prove-long-form-capture-with-a`.
- Bridge review: `scenarios/vrooli-bridge/docs/internal/readiness-review-2026-08-11.html`;
  owner `vrooli-bridge-acknowledged-delivery-idempotent-machine`.
- Web Console review sources: see the maintained
  [review index](../../scenarios/web-console/docs/internal/reviews/INDEX.md).

Read current plan status with `plan-manager plans get <owner>`.
Artifact relocation does not change plan acceptance, regression anchors, or
historical execution outcomes. Plan Manager reference-update receipts accompany
the originals; earlier boundary paths remain historical provenance.

Validation for this cleanup: original-byte hashes, local link non-regression,
manifest retirement references, authored plan/context preservation, and whitespace
checks passed. Knowledge Observatory's scoped docs run passed. Web Console and
Stream of Consciousness Analyzer docs runs still report debt in unchanged source
references, commands, and manifest contracts. The merged ledger passes revision,
retirement, and reference checks, but its representative hybrid-search query still
misses after reindex; a separate focused lexical lookup succeeds. Detailed run
receipts and filed follow-up IDs are in the backup's `validation.json`.

## HTML supplements and scenario history — 2026-09-08

Relocated 30 HTML supplements and two editable-canvas manifests to protected
Plan Manager artifacts. Updated source indexes, navigation, plan references,
and phase instructions. Removed copied July 7 template history from 22 scenario
logs and condensed the largest execution tables; 30 progress files changed.
The template owner's history and scenario-specific handoff sections remain.

Recovery locations are relative to the originating installation's runtime home:

- `plan-artifacts/docs-html-progress-20260908/` retains source files at their
  original repository-relative paths, dispositions, and verification receipts.
- `backups/docs-html-progress-20260908/` retains `originals/` and `manifest.json`
  with exact source hashes. Restore to a separate destination and verify hashes;
  transfer these protected directories with the installation's durable backups.

Relocation preserves historical qualifications and does not establish readiness
or close unresolved work. Plan and phase statuses, authored acceptance boundaries,
and reference/context metadata are retained. Git-diff and boundary entries naming
a removed source remain valid historical deletion coverage. Plan Manager refreshes
its computed baseline and maturity projections when records are updated.

Original-byte preservation, canvas completeness, and changed-document file links
passed verification. Scoped Knowledge Observatory preservation, retrieval, and
reference checks passed for the plan-source and device-identity families. Desktop's
docs phase passed in `20260908-003556-8664af3b`; Web Console's docs phase reported
broader reference debt in `20260908-002810-f96db88d`, filed as
`knw-1788827358069393852`. Optional-flag metadata resets encountered during plan
updates were corrected and filed as `knw-1788827723636449835`.

## Plan-source and root-ledger cleanup — 2026-09-08

Moved the validation-coordination source brief and the professional-integrations
source records into their owning Plan Manager artifact roots. Updated the three
persisted plan references before retiring the repository copies; plan status and
acceptance fields were not changed.

Retired the obsolete root problem ledger and template after preserving exact
copies. The canonical documentation ledger records the disposition rather than
repeating stale January 2025 incidents or obsolete swarm-manager instructions.

Protected artifacts:

| Original | SHA-256 | Protected location |
|---|---|---|
| `docs/architecture/validation-coordination-and-plan-family-supervision-source-brief.md` | `0cbe4d8ecc9d223568b639e7027998f06ae1abfa866b7671752375a30cf5976f` | `plan-artifacts/validation-coordination-plan-family-supervision/source-snapshots/docs/architecture/` |
| `docs/plans/source-artifacts/professional-integrations-monetization-conversation-2026-09-03.md` | `03921f1c99be12a9540c68bdaf661a91df481456ab6612c40ef9e6b987323675` | `plan-artifacts/professional-integrations-monetization-ux/source-artifacts/` |
| `docs/plans/source-artifacts/professional-integrations-monetization-contract-examples.md` | `e68b3570946623bf151b431e84e65285a6abc43fa090e7082826a70200a0bd5c` | `plan-artifacts/professional-integrations-monetization-ux/source-artifacts/` |
| `PROBLEMS.md` | `44674596c913f91591f6e1868d9dd6d0936c6ff2d6fb54bb213dcd9f3c6845e9` | `plan-artifacts/docs-cleanup-20260908-opportunities-1-2/root-ledger/` |
| `PROBLEMS_TEMPLATE.md` | `c3698cbff65db4c5f8bd5a89d33e99af3ae47bad2063321bf1403e9e981b6aaa` | `plan-artifacts/docs-cleanup-20260908-opportunities-1-2/root-ledger/` |
