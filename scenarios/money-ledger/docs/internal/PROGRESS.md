# Progress — Money Ledger

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Entries are appended when work lands, not while it is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |
| 2026-08-13 | codex | in-progress | Implemented the journal, ingestion, position, goals, statements, typed CLI, live-data console pages, fixture-only operator-input importer, mobile/first-run experience journeys, and generated contracts. Evidence: API and CLI suites green; UI 34 files/139 tests green; production build green; experience validation still reports capture/binding drift from the running shell and is not yet a completion claim. |

| 2026-08-15 | codex | baseline | Capability-gap plan Phase 1 anchored. Protected-tree digest at start: `docs/monetization` = `f9fe1bcaba7690f4ed9efbbba999c9135ee9560dc97fdd67c98342e49dec4f1d`; `scenarios/prompt-manager/store/teams/monetization` = `cb80bef5bb254f98a7de6f4383ddd99ed05d2cd9caedfd237f872a328ba13546`. Git Control Tower baseline collection `money-ledger-and-offer-desk-close-the-capability-gaps-and-baseline` passed for `money-ledger` and `offer-desk` at `e94acd34b2b373a98f5d80e4c30561e1707a0b00`. Expected-behavior red bar captured in `TestOperatorInputsImportPreservesPendingAsAbsent`; it demonstrates that non-money operator quantities currently contaminate the journal and is the first implementation target. |

| 2026-08-15 | codex | in-progress | Capability-gap implementation: operator-mode import is reachable through RPC and CLI with dry-run default; thirteen operator paths are classified into eight monetary fields, four measures, and one refused derived rate. Staleness windows and observation dates are reported, stale values are not written, and populated rehearsal evidence shows zero time/hour categories in the journal. Four canonical goal declarations now carry ratio/comparand fields and explicit week/month units. Evidence: `docs/internal/evidence/operator-input-rehearsal.json`; API suite green. |
| 2026-08-15 | codex | done-with-boundary | Final capability-gap validation: requirements validation passed; CLI unit suites passed after binding importer flags to proto field names; live operator import, staleness, goal, position, and board rehearsals passed. Comprehensive run `20260815-065204-650333b7` passed 18/21 phases; only shared `ui-health` timeout and Test Genie experience capture reconciliation remained. Protected-source digests matched the Phase 1 anchors. Planned requirements are enumerated in `PROBLEMS.md`; adoption/cutover was not performed. |
| 2026-08-15 | codex | done-with-boundary | Final audit completed: API/CLI Go suites, golangci-lint, both UI suites/type-check/builds, proto build, requirements validation, and protected-tree digest checks passed. The final Offer Desk rerun `20260815-072053-feebd711` also removed the shared template floor findings; Money Ledger retains the documented Test Genie experience-capture boundary from its comprehensive run. Rehearsal copies were moved recoverably to `/tmp/vrooli-money-offer-rehearsal-20260815-final`. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

| 2026-08-13 | codex | partial | Full implementation and hardening pass: books, append-only journal/reversals/transfers, typed ingestion and fixture-only operator-input import, derived position/goals/statements, typed CLI, live console pages, generated proto/endpoint contracts, viewport-scoped first-run/mobile experience states, and governed dependency/storage/measure declarations. Evidence: lifecycle start healthy; API and CLI Go suites pass; UI 145 tests pass with 90.47% statements, 85.84% branches, 85.47% functions; UI type-check/build/lint pass with two template Fast Refresh warnings; direct `experience-manager spec validate` passes with zero findings after BAS stabilized; fresh full run `20260814-022014-e4bd8193` failed only at the Test Genie `ui-health` provider boundary with no findings payload. Protected-tree end hashes: `docs/monetization` `f9fe1bcaba7690f4ed9efbbba999c9135ee9560dc97fdd67c98342e49dec4f1d`; `scenarios/prompt-manager/store/teams/monetization` `cb80bef5bb254f98a7de6f4383ddd99ed05d2cd9caedfd237f872a328ba13546`. Requirement statuses remain planned pending fresh suite sync and recorded Level 3 drills. |
| 2026-08-13 | scenario initialization | done | Generated from `react-vite`. Authored PRD (7 P0, 6 P1, 5 P2 targets), a four-module requirements registry, the domain map (`books`, `journal`, `ingest`, `position`), DATA including the not-regenerable rule, FLOWS with three topology diagrams, INTEGRATIONS centred on the adapter contract, and the experience contract (5 real pages, 3 journeys). Requirements and experience both validate clean. No code written; the example `notes` domain is untouched. |
| 2026-08-13 | scenario initialization | note | Experience validation against the running UI surfaced two template-level accessibility floor failures (tap-target size, mobile safe area) reproducing in both scenarios. Recorded in PROBLEMS.md; not fixed here — it belongs to the `react-vite` template. |

## Level 3 behavioral-drill evidence — 2026-08-14

The following artifacts were produced against the running local services or by the
scenario's expected-behavior tests. A live CLI/API/UI claim is marked only where the
surface was actually exercised; fixture tests are not presented as end-to-end proof.

| Drill | Artifact and result | Boundary still open |
|---|---|---|
| Manual entry is first-class | `money-ledger ingest event-ingest --json` wrote `drill-sale-1` with `BASIS_OPERATOR_ASSERTED`; `ledger journal-list --json` returned the same typed row. | No upstream adapter was needed for this manual-path proof. |
| A correction cannot be an edit | `money-ledger ledger journal-reverse --json` created `9acbd76a-d330-4791-8871-bca9e7962dc1` with `reversal_of=1aa3be2a-43c8-40b7-8981-c8baf7ba6138`; `journal-list --json` returned both entries. Backend proof is also `TestStoreIngestIsIdempotentAndReversalIsAppendOnly`. | A dedicated live browser scan for edit/delete affordances remains absent; the UI state suite covers the authored journal states. |
| Ingestion is idempotent | Repeating the same `event-ingest --json` returned `duplicate=true` and `skipped_duplicates=1`; backend proof is the same named test. | — |
| Position degrades legibly | `money-ledger ingest adapter-run --adapter-id drill-manual --json` returned `status=failed`, a named reason, and `last_success_at`; `position-show --json` returned `partial=true` with no synthesized zero. Backend proof is `TestFailedAdapterIsVisibleAndNeverWritesZero`; UI fixture proof is `names an unavailable adapter and its impact [REQ:POS-004]`. | The UI artifact is fixture-based, not a browser read of this exact live outage. |
| Pending operator input is absent, not zero | `TestOperatorInputsImportPreservesPendingAsAbsent` passed; `TestOperatorInputsFixtureImportCarriesSourceProvenance` passed. | The live fixture import is not exposed as a safe operator CLI workflow. |

The remaining joint drills are recorded by Offer Desk because their primary claim is
the offer lifecycle or board; the paired evidence is in that scenario's progress log.
The full suite run `20260814-031020-841b7ec6` was still server-owned at the time of
this entry and its UI-health execution phase remains an infrastructure boundary.

| 2026-08-14 | codex | done-with-boundary | Final validation: fresh comprehensive run `20260814-034428-59a384b3` completed 20/21 phases; portability, structure, contracts, API, storage, workflow, experience, unit, and requirements-linked evidence passed. The sole failure is the shared Test Genie `ui-health` execution provider, which returns no findings payload and times out at the provider boundary; direct static-only UI-health has zero required findings, and direct experience validation passes with zero findings. Requirements validation is PASSED at L3 for PRD contract, registry, intent linkage, and evidence traceability; the authoritative matrix is 16 complete and 10 explicitly planned. Final Level 3 drill evidence is recorded above. Protected-tree hashes are unchanged from the recorded end values. |
| 2026-08-15 | codex | done-with-boundary | Post-fix authoritative run `20260815-092007-13e9dfb6` completed 20/21 phases: experience passed with zero findings after aligning page-specific empty-state selectors and the journal form semantics; the sole failure remains Test Genie `ui-health` with `failure_class=missing_dependency` and zero findings, even after `ui-health`, `code-facts`, and qdrant were started healthy through the control plane. UI tests passed 35 files/145 tests; type-check and production build passed. |
