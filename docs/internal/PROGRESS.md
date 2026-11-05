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

## Nutrition planner full-day Today slots — 2026-09-18

Extended the planning client and Today surface to request and display configured
breakfast, lunch, dinner, and open snack slots. Open slots remain explicitly
user-controlled; unresolved required slots remain visible without relaxing rules.
Added client and UI regressions for meal-slot encoding and full-day rendering.
UI validation passed with 34 files and 163 tests; coverage remained above the
configured threshold at 89.77% statements and 85.01% branches.

## Nutrition planner portability and transfer center — 2026-09-18

Completed the portability slice for native recipe export/restore, including
canonical yield, serving unit, ingredients, and preparation-map data. Added
transfer-center cards for recipe JSON, workspace backup, weekly PDF, grocery
CSV, and recipe PDF exports, plus staged restore and explicit omission/error
messaging. Backend portability tests, endpoint drift validation, and the
server-owned unit/API/storage run passed (`20260918-220119-cf66bf7c`). UI
validation passed with 35 files and 170 tests at 94.81% statements and 85.88%
branches. Added explicit A4/US Letter selection to server-side recipe and
weekly PDF generation, with validated page-size input and page-count footers;
generated proto clients were refreshed and portability regressions cover Letter
output and invalid format rejection. Existing Test Genie advisories remain
non-blocking, including the pre-existing injectable-seam and low-coverage
findings.

Browser Automation Studio captures also reached the live scenario successfully
at mobile and desktop surfaces: Today mobile (`bd08663e-ebc7-4d01-829c-ab700c62cab9`),
Week desktop (`e22b8aeb-cc75-40d9-b024-ed3af60ca021`), setup mobile
(`bf536041-5605-4670-8a19-c9bb20abb004`), and transfer at both desktop
(`df13920b-8cf4-445f-958b-278ebf550cfd`) and mobile
(`29a6ada6-4c58-43d0-89ae-3d047e8a9f9b`) all returned ready snapshots without
capture errors.

Closed a concrete accessibility gap in Today and Week swap surfaces by adding
shared dialog focus containment, Escape cancellation, focus restoration, and
keyboard regressions. The transfer center retains its axe-clean regression;
focused Today/Week validation now passes 19 tests.

Reconciled the portability contract after validation: recipe exports now report
only nutrition and cost as omissions, and staged native imports enforce the
documented 200-recipe, 80-ingredient-use, 30-step, and 10,000-character note
limits. Added regressions for the step and note limits; portability package
tests remain green.

Post-reconciliation validation passed: all Go packages, endpoint generation
with no manifest drift, UI TypeScript, 35 UI test files / 173 tests, and the
configured UI coverage gate (93.93% statements, 85.48% branches). The focused
playbook run reached execution and found only the shared pre-existing BAS
observer-content safety errors in performance example files.

Closed the native recipe-import gap in the transfer center. Added preview and
apply RPCs with duplicate/conflict counts, explicit keep/copy/replace policy,
workspace revision barriers, atomic SQLite writes, idempotency replay, and
remapped IDs for copied conflicts. The UI now auto-detects `daily.recipes`,
stages the review, and routes the selected policy to apply. Targeted API/UI
regressions, TypeScript, all Go packages, and endpoint-drift validation pass.

Preserved known allocated cost when stock availability is unknown, while
keeping checkout cost and completeness explicitly unresolved. `CalculateMany`
now aggregates each independently known cost component instead of discarding
known values because another item is unresolved; added R1 cost regressions.

Raised the nutrition surface with persisted supplement schedule controls: create
confirmed schedules, list them, and pause/resume through revision-checked
updates. The UI states that schedules apply only on confirmed applicable dates
and never rewrite prior intake; targeted nutrition UI tests and TypeScript pass.

### 2026-09-19 — Accepted calendar allocation slice

Added the first durable schedule boundary to `personal-planner`: a generated
CalendarService with SQLite-backed, non-overlapping accepted allocations,
measured daily capacity, and work-item enrichment. Plan now reads and places
accepted allocations; Today reads the same calendar read model; Review measures
planned minutes alongside recorded focus without inferring completion. API and
UI tests pass, the managed runtime was restarted healthy (optional
`notification-hub` remains degraded), and live Connect smoke created and read
an allocation at 10:00 for 45 minutes. The CLI intentionally omits the new
calendar procedures until its date/overlap interaction is designed. Plan now
also exposes a durable accepted-placement form for choosing work item, start,
and duration; overlap rejection remains explicit. The same Calendar record set
now has an inclusive local-date range read used by Plan's Week and Agenda
projections; no client-only schedule data is invented.

### 2026-09-19 — Observatory checkpoint and durable Today focus

The 1585×992 Day/Night checkpoint was captured and compared against the
approved mockups. The shell was corrected from the generic wide
“Personal Planner/Home” treatment to a narrow Planner/Observatory rail with
Today navigation, editorial content width, and scene/card geometry closer to
the oracle. Today’s primary focus control now calls the durable FocusService
start/pause path and reports failed transitions without local fiction. UI
coverage remains above all configured floors (95.13% statements, 86.20%
branches, 86.72% functions; 132 tests). The visual gate still has a bounded
responsive/accessibility sweep outstanding.

### 2026-09-19 — Calendar range projection and Plan views

Extended the accepted Calendar record set with an explicit inclusive local-date
range RPC. Plan now offers Day, Week, and Agenda projections over that shared
server data, with no client-only schedule records. Endpoint parity, CLI omission
governance, API tests, UI coverage (95.13% statements, 86.20% branches,
86.72% functions; 131 tests), live range read, and Test Genie unit/contracts/
experience run `20260919-073604-28dfc4f6` all pass. The CLI evidence artifact was
regenerated after the manifest changed.

### 2026-09-19 — Settings product boundary and honest monetization story

Extended Settings with a visible, localized product promise: the current
personal planner core is included, local preferences remain private by default,
and optional provider connections, forecasts, and assistance are future
hypotheses rather than an active upgrade path. The UI presents no billing,
payment, or non-functional upgrade controls. The English rendering regression
and full UI suite pass: 29 files / 133 tests with 95.19% statements, 86.64%
branches, and 86.72% functions. This is evidence for the product boundary, not
a claim that deferred availability, integrations, learning, or billing domains
are complete.

### 2026-09-19 — Workspace planning profile vertical slice

Added the first real workspace boundary behind Settings. The generated
WorkspaceService persists a revisioned planning profile with IANA timezone,
week-start policy, daily capacity, protected reserve, and preferred focus
session length. Settings loads and saves those values through Connect with
localized controls; the CLI exposes `workspace profile` and
`workspace update-profile`. API, CLI, endpoint-parity, TypeScript, locale,
and UI coverage checks pass. Live read/update/read smoke changed the profile
from UTC/480 minutes to America/New_York/420 minutes at revision 2, and the
CLI reported the same persisted values. Test Genie
`20260919-081349-f61bca55` passed unit, contracts, and experience. Availability
weekday intervals, exceptions, provider projections, and capacity integration
remain deliberately open.

The same profile now feeds Calendar's Today capacity read model: accepted
schedule capacity subtracts the protected reserve, so Settings and Today no
longer have competing availability assumptions. A SQLite regression asserts
420 minutes capacity minus 60 reserve yields 360 available minutes and correct
breathing room.

### 2026-09-19 — Observatory surface consistency and runtime validation

Applied semantic theme tokens across Plan, Goals, Focus, and Review secondary
surfaces, migrated Review's weekly activity table to the shared DataTable, and
moved the Today shared Dialog adapter into `ui/src/components/PlannerDialog.tsx`.
Today capture now keeps a failed save honest and the accessibility regression
waits for the shared overlay's exit lifecycle. Type-check/build passed, and
Test Genie `20260919-191234-ffb49c1d` passed ui-health at L5 across all seven
routes. Remaining findings are advisory: clipped short labels, existing raw
palette literals, and the unused health component; a whole-file Dashboard test
mock remains flaky while the isolated Dashboard suite passes.

Focused Test Genie run `20260919-082216-446aa291` passed unit, contracts, and
experience after this integration. The latest comprehensive run
`20260919-081553-ed35b988` failed on repository-wide maturity/security findings;
that failure is retained as a limitation rather than represented as a green
scenario certification.
