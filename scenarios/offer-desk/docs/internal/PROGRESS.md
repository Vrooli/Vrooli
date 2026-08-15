# Progress — Offer Desk

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Entries are appended when work lands, not while it is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |
| 2026-08-13 | codex | in-progress | Implemented the offer graph, lifecycle gates, multi-clause evaluation, typed CLI, live board/catalog/gate console data reads, fixture-only catalog importer, mobile/first-run experience journeys, and generated contracts. Evidence: API and CLI suites green; UI 34 files/139 tests green; production build green; experience validation still reports capture/binding drift from the running shell and is not yet a completion claim. |

| 2026-08-15 | codex | baseline | Capability-gap plan Phase 1 anchored. Protected-tree digest at start: `docs/monetization` = `f9fe1bcaba7690f4ed9efbbba999c9135ee9560dc97fdd67c98342e49dec4f1d`; `scenarios/prompt-manager/store/teams/monetization` = `cb80bef5bb254f98a7de6f4383ddd99ed05d2cd9caedfd237f872a328ba13546`. Git Control Tower baseline collection `money-ledger-and-offer-desk-close-the-capability-gaps-and-baseline` passed for `money-ledger` and `offer-desk` at `e94acd34b2b373a98f5d80e4c30561e1707a0b00`. Expected-behavior red bar captured in `TestImportedStatusRecognizesCorpusShapes` and `TestUnrecognizedStatusIsNotSilentlyIdea`; implementation work follows. |

| 2026-08-15 | codex | in-progress | Operator catalog rehearsal against a 44-file copy completed: 44 files reported, 28 lifecycle/tier/membership records written, four variants, six membership edges, three requires edges, and eight sells_at edges. Malformed status and bidirectional manifest-drift copies both produced blocking findings with apply false. The typed Space RPC/CLI and Money Ledger board wiring are live. Evidence: `docs/internal/evidence/catalog-rehearsal.json`; API and UI suites green. BAS observer cases were scaffolded for every active page so the next scenario-owned run can capture reconciliation evidence. |
| 2026-08-15 | codex | done-with-boundary | Final capability-gap validation: requirements validation passed; CLI unit suites passed after binding importer flags to proto field names; live catalog import, malformed/drift rejection, board, projection, evaluation, and dependency-degradation rehearsals passed. Comprehensive run `20260815-070211-6759100e` passed 19/21 phases; only shared `ui-health` timeout and Test Genie experience capture reconciliation remained. Protected-source digests matched the Phase 1 anchors. Planned requirements and the cross-scenario journey backlog item are enumerated in `PROBLEMS.md`; adoption/cutover was not performed. |
| 2026-08-15 | codex | done-with-boundary | Final post-floor-fix comprehensive rerun `20260815-072053-feebd711` passed 20/21 phases. `ui-health` rendered successfully with advisory-only debt, and the template mobile tap-target/safe-area floor findings are gone. The sole remaining phase failure is experience capture reconciliation (18 unresolved bindings, 20 unjoined captures, 16 unproven claims); no source selectors were changed to hide it. Requirements, CLI/API/UI/proto checks, live rehearsals, and protected-source digests are clean; adoption/cutover remains out of scope. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

| 2026-08-13 | codex | partial | Full implementation and hardening pass: typed offer graph, audited lifecycle transitions, trigger/fact evaluation, operator-only promotion, fixture-only catalog importer, Money Ledger actuals/posture board, typed CLI projection, live console pages, generated proto/endpoint contracts, viewport-scoped first-run/mobile experience states, and governed dependency/storage/measure declarations. Evidence: lifecycle start healthy; API and CLI Go suites pass; UI 146 tests pass with 90.38% statements, 85.67% branches, 89.18% functions; UI type-check/build/lint pass with two template Fast Refresh warnings; docs, storage, and workflow phases passed in `20260814-020150-91fd1b19`; direct `experience-manager spec validate` passes with zero findings after BAS stabilized; fresh full run `20260814-022717-0abef08d` terminated with only the Test Genie `ui-health` provider failing and no findings payload. Protected-tree end hashes: `docs/monetization` `f9fe1bcaba7690f4ed9efbbba999c9135ee9560dc97fdd67c98342e49dec4f1d`; `scenarios/prompt-manager/store/teams/monetization` `cb80bef5bb254f98a7de6f4383ddd99ed05d2cd9caedfd237f872a328ba13546`. Requirement statuses remain planned pending fresh suite sync and recorded Level 3 drills. |
| 2026-08-13 | scenario initialization | done | Generated from `react-vite`. Authored PRD (7 P0, 5 P1, 4 P2 targets), a five-module requirements registry, the domain map (`catalog`, `gates`, `board`), DATA, FLOWS with lifecycle and topology diagrams, INTEGRATIONS, and the experience contract (4 real pages, 2 journeys). Requirements and experience both validate clean. No code written; the example `notes` domain is untouched. |
| 2026-08-13 | scenario initialization | note | Experience validation against the running UI surfaced two template-level accessibility floor failures (tap-target size, mobile safe area) reproducing in both scenarios. Recorded in PROBLEMS.md; not fixed here — it belongs to the `react-vite` template. |

## Level 3 behavioral-drill evidence — 2026-08-14

| Drill | Artifact and result | Boundary still open |
|---|---|---|
| The unreachable status becomes reachable | Expected-behavior test `TestSchedulerPromotesSatisfiedCandidateWithoutManualEvaluate` passed with a fake ticker and exactly one evaluation. The live CLI path also reached `TRIGGER_MET` after `gates-evaluate --json`, with the evaluation naming `drill_revenue` and its observation age. | The live CLI transcript used an explicit evaluate call; a minute-cadence scheduler transition has not yet been captured as a separate live artifact. |
| Unknown is not false | `TestLifecycleReachesTriggerMetAndUnknownIsNotFalse` and `TestStaleFactIsUnknownAndLeavesCandidateInPlace` passed; they assert UNKNOWN, explanation/gap, and unchanged candidate state. | No separate live CLI run for a missing fact and a stale fact has been captured. |
| A refusal teaches | Live `catalog-transition --status CANDIDATE --json` refused with `candidate_requires_trigger` and remediation. UI proof is `renders a refusal as an explicit error with its remediation [REQ:UI-001] [REQ:UI-002]`; API proof is `TestCandidateRequiresTriggerAndPromotionIsOperatorOnly`. | The three-surface artifact is split across a live CLI response plus API/UI tests, not one live API/CLI/UI recording. |
| An agent cannot promote | Live `gates-promote --role agent --json` returned an operator-only proposal and `catalog-list --json` left the node non-active; backend proof is `TestCandidateRequiresTriggerAndPromotionIsOperatorOnly`. | — |
| The board degrades legibly | Live `board-show --json` returned a catalog row while preserving `money-ledger` and `money-ledger.actuals` availability reasons; backend proof is `TestBoardReportsLedgerUnavailableWithoutInventingActuals`. | This used an unconfigured ledger client rather than stopping the already-running Money Ledger process. |
| The pair's headline claim | After operator promotion, live `board-show --json` returned status `ACTIVE`, rank reason `active and earning nothing`, and source attribution for unavailable actuals. | No screenshot/screen recording was captured. |
| The importer is honest | `TestImportTreeReportsBrokenReferencesWithoutCopyingNarrative` passed with two files read/written, one finding, and narrative excluded; `MIG-001`/`MIG-002` refs resolve. | Manual source-retirement review remains intentionally planned; no source was deleted. |

The Money Ledger progress log records the paired manual-entry, correction,
idempotency, position-degradation, and pending-operator artifacts. The full Offer
Desk run `20260814-031220-2a348d91` is server-owned; the Test Genie UI-health provider
boundary is not represented as a product finding. Requirement statuses remain
planned until the authoritative full runs sync them.

| 2026-08-14 | codex | done-with-boundary | Final validation: fresh comprehensive run `20260814-035300-bf1ee069` completed 20/21 phases; portability, structure, contracts, API, storage, workflow, experience, unit, and requirements-linked evidence passed. The sole failure is the shared Test Genie `ui-health` execution provider, which returns no findings payload and times out at the provider boundary; direct static-only UI-health has zero required findings, and direct experience validation passes with zero findings. Requirements validation is PASSED at L3 for PRD contract, registry, intent linkage, and evidence traceability; the authoritative matrix is 18 complete and 6 explicitly planned. Final Level 3 drill evidence is recorded above. Protected-tree hashes are unchanged from the recorded end values. |
| 2026-08-15 | codex | done-with-boundary | Post-fix authoritative run `20260815-090238-1e5e8a8f` completed 20/21 phases: experience passed with zero findings after aligning the graph binding to the normalized `image` role; the sole failure remains Test Genie `ui-health` with `failure_class=missing_dependency` and zero findings. Offer Desk UI tests/type-check/build passed, and the shared ui-health dependency chain was later started healthy through the control plane. |
