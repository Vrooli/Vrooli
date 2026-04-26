# Plan: Decouple workspace-sandbox locking from acceptance filtering

## Purpose

Restore the locked auditability contract's invariant that `NoLock` and acceptance evaluation are orthogonal concerns. Today, `evaluateAcceptance` in `scenarios/workspace-sandbox/api/internal/sandbox/service.go` short-circuits to `AcceptanceStatusAccepted` whenever `sandbox.NoLock` is true, silently bypassing every configured allow/deny rule. This fix removes that shortcut so acceptance allow/deny rules apply identically regardless of `NoLock`, while leaving locking semantics (mutual exclusion / scope reservation) untouched.

This is the foundation fix that unblocks `execute/agent-manager-sandbox-auto-apply-defaults` and `execute/sandbox-runtime-e2e-verification` in the `agent-sandbox-audit-foundation` initiative.

## Greenfield Declaration

**This is NOT a greenfield change.** It modifies behavior in an existing call path (`evaluateAcceptance`) without introducing a compatibility shim. Per Decision 2 (no migration), the new behavior takes effect immediately on deploy: any existing in-flight `NoLock=true` sandbox that next calls `Diff`/`Apply` will see acceptance rules re-evaluated. The prior behavior was a contract violation, so no grandfathering, dual-mode, or `legacy_accept_all` flag is added. Future contributors must not re-introduce a `NoLock`-conditional bypass under any name.

## Required Reading

Before executing this plan, the implementing agent should read the following skills (in order):

1. `prompt-manager skill read scientific-debugging` — kind-required for fix items. Hypothesis-driven root-cause framing, regression test discipline.
2. `prompt-manager skill read assumption-mapping-and-hardening` — this fix is exactly the removal of an incorrect implicit assumption ("noLock implies accept-all"). The skill's "Soften or Remove Fragile Assumptions" guidance applies directly to the patch and the new tests.
3. `prompt-manager skill read test` — guidance on assertion quality and high-signal regression coverage. Apply to the new acceptance tests under `scenarios/workspace-sandbox/api/internal/sandbox/**`.

## Source-of-Truth References

- **Contract**: `scenarios/swarm-manager/research/agent-sandbox-auditability-contract/conclusion.md` Finding 4 M1.
- **Contract mirror**: `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` — already documents that this fix removes the historical noLock-implies-accept-all shortcut.
- **Initiative neighborhood**: `agent-sandbox-audit-foundation`. Sibling members:
  - `research/agent-sandbox-auditability-contract` (completed) — produced the contract this fix executes against.
  - `execute/agent-manager-sandbox-auto-apply-defaults` (depends on this fix) — wires the writer side. **Do not** add `state=pending-review` provenance writes in this fix; that lives in `execute/gct-pending-ai-provenance-hardening` (upstream initiative `git-control-tower-ai-provenance`). This fix only removes the bypass; out-of-acceptance changes remain `AcceptanceStatusIgnored` per the existing behavior of `filterChangesByAcceptance`.
  - `execute/sandbox-runtime-e2e-verification` (downstream) — owns the end-to-end multi-sandbox-same-scope verification at the spawn-surface layer. This fix only owns the unit + assumption-test layer.

## Problem Statement

`scenarios/workspace-sandbox/api/internal/sandbox/service.go:1145-1152` short-circuits `evaluateAcceptance` when `sandbox.NoLock` is true and unconditionally returns `AcceptanceStatusAccepted`. This conflates two orthogonal concerns:

1. **Locking** — `NoLock` is meant to control mutual exclusion / scope reservation only (see `pathutil.go:110` and `service.go:437,542`).
2. **Acceptance** — allow/deny rules are meant to gate apply eligibility per file.

The conflation means any sandbox created with `NoLock=true` silently bypasses configured acceptance rules. Per the locked auditability contract, locking and acceptance are independent: `NoLock` controls only mutual exclusion, and acceptance allow/deny is evaluated against every candidate apply regardless of `NoLock`.

## Scope (In / Out)

**In scope:**
- Remove the `NoLock`→accept-all shortcut in `evaluateAcceptance`.
- Update `DefaultNoLock` doc comment so future readers know it does not affect acceptance.
- Update `AUDITABILITY_CONTRACT.md` to past tense post-merge.
- Add inline code comment at the removed bypass site referencing `AUDITABILITY_CONTRACT.md` Finding 4 M1.
- Add regression coverage at the unit + service-integration layer.
- Add an investigation-flow assumption test pinning that investigation runs do not produce sandbox file changes (per Decision 1).

**Out of scope:**
- Adding `state=pending-review` provenance for out-of-acceptance changes — `execute/gct-pending-ai-provenance-hardening`.
- Adding `runOutcome` to provenance — same upstream item.
- Wiring run-end auto-apply defaults — `execute/agent-manager-sandbox-auto-apply-defaults`.
- End-to-end verification at spawn-surface layer — `execute/sandbox-runtime-e2e-verification`.
- Changing `DefaultNoLock` (currently `true`) — already aligned with the contract's `lock=false` default; only docs need clarifying.
- Migrating existing `no_lock=true` rows in the sandbox table (Decision 2: no migration).

## Scope (Acceptance Globs)

- `acceptance_allow`: `scenarios/workspace-sandbox/**`, `scenarios/agent-manager/**` (already set).
- `acceptance_deny`: none required.

Planned changes by glob:
- `scenarios/workspace-sandbox/api/internal/sandbox/service.go` — remove the bypass branch in `evaluateAcceptance`; add inline comment referencing the contract.
- `scenarios/workspace-sandbox/api/internal/sandbox/service_test.go` — new unit + service-integration tests.
- `scenarios/workspace-sandbox/api/internal/sandbox/assumptions_test.go` — back-compat (empty acceptance config) assumption test.
- `scenarios/workspace-sandbox/api/internal/config/config.go` — clarify `DefaultNoLock` doc comment.
- `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` — flip the historical reference to past tense post-merge.
- `scenarios/agent-manager/api/internal/orchestration/investigation.go` — **no production code change** (Decision 1A); only a new test colocated with investigation tests asserting investigation runs do not produce sandbox file changes.

## Current Technical Context

| File | Lines | Role |
|---|---|---|
| `scenarios/workspace-sandbox/api/internal/sandbox/service.go` | 1145-1152 | The bypass branch in `evaluateAcceptance` to be removed. |
| `scenarios/workspace-sandbox/api/internal/sandbox/service.go` | 856 | `Diff` call site for `evaluateAcceptance` (must continue to work). |
| `scenarios/workspace-sandbox/api/internal/sandbox/service.go` | 1341 | `filterChangesByAcceptance` call site for `evaluateAcceptance` (must continue to work). |
| `scenarios/workspace-sandbox/api/internal/sandbox/service.go` | 437, 542 | Reservation-path lock branches keyed on `NoLock`. **Do not touch.** |
| `scenarios/workspace-sandbox/api/internal/sandbox/pathutil.go` | 110 | `FindConflicts` skips no-lock sandboxes. **Do not touch.** |
| `scenarios/workspace-sandbox/api/internal/config/config.go` | 210-216 | `DefaultNoLock` default + doc comment. Doc comment to be clarified. |
| `scenarios/workspace-sandbox/api/internal/sandbox/service_test.go` | — | Service-level integration test harness. New tests land here. |
| `scenarios/workspace-sandbox/api/internal/sandbox/assumptions_test.go` | — | Assumption-test surface. Back-compat case lands here. |
| `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` | line 40 | Existing one-line note about this fix; update to past tense post-merge. |
| `scenarios/agent-manager/api/internal/orchestration/investigation.go` | 433 | Sets `NoLock: true` on investigation runs. No production change; new test asserts investigations do not write files. |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go` | 1724 | `noLockFromSandboxConfig` returns `nil` when `SandboxConfig.NoLock` is false; remains correct after the fix. |

## Target End State

1. `evaluateAcceptance(sandbox, path)` returns identical results for `NoLock=true` and `NoLock=false` given the same acceptance config and file path.
2. `NoLock` controls only mutual exclusion / scope reservation (`pathutil.go`, reservation-path branches in `service.go`).
3. Multiple concurrent sandboxes over the same scope, each with `NoLock=true` and distinct acceptance configs, evaluate acceptance independently.
4. Agent-manager investigation runs continue to function with `NoLock: true` (concurrency preserved) and produce no sandbox file changes (verified by new test).
5. `AcceptanceInfo.Status` for out-of-allow files surfaces as `AcceptanceStatusIgnored`; matched-deny files surface as `AcceptanceStatusDenied`. (Mapping `Ignored`/`Denied` → `pending-review`/`denied` is `execute/gct-pending-ai-provenance-hardening`'s job.)
6. Sandboxes with no configured acceptance rules continue to accept all changes (back-compat preserved via the existing `isCriteriaEmpty` paths in `evaluateAcceptance`).
7. `DefaultNoLock` doc comment explicitly disclaims any influence on acceptance.
8. Inline code comment at the (removed) bypass site references `AUDITABILITY_CONTRACT.md` Finding 4 M1.
9. `AUDITABILITY_CONTRACT.md` line 40 reads in past tense ("was removed by …").

## Implementation Strategy

### Phase 1 — Code change (workspace-sandbox)

1. In `scenarios/workspace-sandbox/api/internal/sandbox/service.go`, remove the `if sandbox.NoLock { return AcceptanceInfo{Status: AcceptanceStatusAccepted, ...} }` short-circuit at lines 1145-1152.
2. The function now falls through to its existing allow/deny evaluation, which already accepts everything when both criteria are empty (`isCriteriaEmpty(acceptance.Allow)` → matched allow rules), so back-compat for sandboxes with no configured acceptance rules is preserved.
3. Add an inline comment at the deleted site referencing `AUDITABILITY_CONTRACT.md` Finding 4 M1 so future contributors understand the historical shape and the contract that forbids re-introducing it (per Decision 4A).
4. Leave `NoLock`-driven branches in `pathutil.go:110` and `service.go:437,542` untouched — those remain correct lock-semantics behavior.

### Phase 2 — Config doc clarification

5. In `scenarios/workspace-sandbox/api/internal/config/config.go`, update the `DefaultNoLock` doc comment (lines 210-216) to add an explicit sentence: *"DefaultNoLock controls only mutual exclusion / scope reservation. It has no effect on acceptance evaluation; acceptance allow/deny rules are applied independently."*

### Phase 3 — Agent-manager investigation handling (Decision 1A)

6. **No production code change to `investigation.go`.** Investigations remain `NoLock: true` for concurrency. Add an assumption test colocated with investigation tests asserting investigation runs produce no sandbox file changes (so even if acceptance now matters for them in principle, it is moot in practice). This pins the right invariant regardless of this fix.

### Phase 4 — Regression tests (Decision 3B)

7. Land focused unit tests in `service_test.go` covering the four primary cases:
   - `NoLock=true` + allow rule + out-of-allow path → `AcceptanceStatusIgnored`
   - `NoLock=true` + deny rule + matched-deny path → `AcceptanceStatusDenied`
   - `NoLock=true` + non-empty acceptance config: equivalence to `NoLock=false` for the same input
   - Both `Diff` (line 856) and `filterChangesByAcceptance` (line 1341) call sites exercise the new behavior
8. Land a back-compat assumption test in `assumptions_test.go`: `NoLock=true` + empty acceptance config → `AcceptanceStatusAccepted`.
9. Land a service-level integration test in `service_test.go` that creates two real sandboxes via the Service layer with overlapping scope, both `NoLock=true`, with distinct acceptance configs, and verifies each evaluates acceptance independently.

### Phase 5 — Documentation (Decision 4A)

10. In a follow-up commit on the same branch (post-merge of code change), update `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` line 40 to past tense ("was removed by `fix/workspace-sandbox-lock-and-acceptance-semantics`").

## Contract Decisions

The following decisions were resolved in workshop round 1 and are now load-bearing:

| ID | Topic | Resolution | Source |
|---|---|---|---|
| D1 | Agent-manager investigation flow handling | **A** — Leave `investigation.go` unchanged (`NoLock: true` preserved for concurrency); add an investigation-flow test asserting investigation runs produce no sandbox file changes. | Round 001 |
| D2 | Migration of existing in-flight `NoLock=true` sandboxes | **A** — No migration. Behavior changes immediately on deploy. The prior behavior was a bug; sandboxes with no configured acceptance still see all-accepted via the existing `isCriteriaEmpty` paths. No `legacy_accept_all` column, no warn-only mode. | Round 001 |
| D3 | Test surface and depth of regression coverage | **B** — Four focused unit tests in `service_test.go` + one back-compat case in `assumptions_test.go` + one service-level integration test for the multi-sandbox-same-scope case. No agent-manager → workspace-sandbox e2e test (deferred to `execute/sandbox-runtime-e2e-verification`). | Round 001 |
| D4 | Documentation surface for "do not reintroduce" guidance | **A** — Inline code comment at the removed bypass site referencing `AUDITABILITY_CONTRACT.md` Finding 4 M1, plus update `AUDITABILITY_CONTRACT.md` to past tense post-merge. No `PROBLEMS.md` entry. | Round 001 |

Additional contract notes from round 1 info items:
- `noLockFromSandboxConfig` at `run_executor.go:1724` continues to return `nil` when `SandboxConfig.NoLock` is false, deliberately letting workspace-sandbox apply its own `DefaultNoLock` (true). This stays correct after the fix.
- This is a greenfield test surface in the sense that no existing tests in `service_test.go` or `assumptions_test.go` reference NoLock + acceptance interactions today. The new tests become the canonical coverage for the orthogonality contract.

## Testing Plan

Unit, assumption, and integration tests live under `scenarios/workspace-sandbox/api/internal/sandbox/`. Investigation-flow test lives under `scenarios/agent-manager/api/internal/orchestration/`.

| # | Behavior | File | Layer | Why this test |
|---|---|---|---|---|
| 1 | `NoLock=true` + allow rule + out-of-allow path → `Ignored` | `scenarios/workspace-sandbox/api/internal/sandbox/service_test.go` | Unit | Pins the primary regression — what the bypass used to mask. |
| 2 | `NoLock=true` + deny rule + matched-deny path → `Denied` | `scenarios/workspace-sandbox/api/internal/sandbox/service_test.go` | Unit | Pins that deny rules also apply under `NoLock`. |
| 3 | `NoLock=true` + non-empty acceptance: equivalence to `NoLock=false` | `scenarios/workspace-sandbox/api/internal/sandbox/service_test.go` | Unit | Pins the orthogonality invariant directly. |
| 4 | `Diff` (line 856) and `filterChangesByAcceptance` (line 1341) both reach the new behavior under `NoLock=true` | `scenarios/workspace-sandbox/api/internal/sandbox/service_test.go` | Unit | Covers both call sites of `evaluateAcceptance`. |
| 5 | `NoLock=true` + empty acceptance config → `Accepted` | `scenarios/workspace-sandbox/api/internal/sandbox/assumptions_test.go` | Assumption | Documents and enforces the back-compat assumption. |
| 6 | Two concurrent sandboxes, overlapping scope, both `NoLock=true`, distinct acceptance configs → independent evaluation | `scenarios/workspace-sandbox/api/internal/sandbox/service_test.go` (or new `multi_sandbox_test.go`) | Service-integration | Honors the spec's explicit multi-sandbox-same-scope callout. |
| 7 | Investigation run produces no sandbox file changes | `scenarios/agent-manager/api/internal/orchestration/investigation_test.go` (or colocated) | Assumption | Pins the right invariant for Decision 1A — investigations are read-only by design. |

Run via:
```bash
cd scenarios/workspace-sandbox && make test
cd scenarios/agent-manager && make test
```
or equivalently `vrooli scenario test workspace-sandbox` and `vrooli scenario test agent-manager`.

## Rollout / Validation Checklist

1. **Code & tests in place:**
   - [ ] Bypass branch removed from `service.go:1145-1152` with inline comment referencing `AUDITABILITY_CONTRACT.md` Finding 4 M1.
   - [ ] `DefaultNoLock` doc comment updated in `config.go`.
   - [ ] All seven tests above land and pass.
2. **Local validation:**
   - [ ] `cd scenarios/workspace-sandbox && make test` passes.
   - [ ] `cd scenarios/agent-manager && make test` passes (investigation flow regression check).
   - [ ] `golangci-lint run` passes for both scenarios.
   - [ ] `gofumpt -l scenarios/workspace-sandbox scenarios/agent-manager` reports no diffs.
3. **Restart and smoke-check the running stack** (per workspace-sandbox lifecycle guidance):
   - [ ] `vrooli scenario restart workspace-sandbox`
   - [ ] `vrooli scenario restart agent-manager`
   - [ ] Verify both come up healthy via `vrooli scenario logs workspace-sandbox` and `vrooli scenario logs agent-manager`.
4. **Post-merge documentation flip:**
   - [ ] Update `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` line 40 to past tense ("was removed by `fix/workspace-sandbox-lock-and-acceptance-semantics`").
5. **Final cleanup:**
   - [ ] `vrooli scenario restart workspace-sandbox && vrooli scenario restart agent-manager` to leave the running stack on the new code.

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Investigation runs (`investigation.go:433`) silently break because they relied on the bypass to write something we didn't realize they wrote. | Low | Medium | Decision 1A: keep `NoLock: true`, add an investigation-flow test asserting investigation runs produce no sandbox file changes. Investigations are read-only by design (event-timeline + diff context attachments). |
| Existing in-flight sandboxes in the DB with `NoLock=true` and configured acceptance rules see different diff/apply behavior post-deploy than they saw at create-time. | Low | Low | Decision 2A: no migration. The prior behavior was a contract violation; sandboxes with no acceptance config are unaffected; sandboxes with explicit acceptance config now get the correct behavior. |
| Contributors re-introduce the bypass during future cleanup (the comment at line 1146 made the bypass look intentional). | Medium | Medium | Decision 4A: inline comment at the removed site + AUDITABILITY_CONTRACT.md update + regression tests. Constraint discoverable from three angles. |
| `DiffWithAcceptance` and other downstream consumers of `evaluateAcceptance` change behavior in subtle ways (e.g., diff result composition shifts because more files now report `Ignored`/`Denied`). | Medium | Low | Test #4 covers both call sites (line 856 in `Diff`, line 1341 in `filterChangesByAcceptance`). UI consumers already render `AcceptanceStatusIgnored`/`Denied` distinctly. |
| Cross-initiative implication — GCT review-queue work (`execute/gct-pending-ai-provenance-hardening`) assumed it would receive `state=pending-review` for out-of-acceptance changes; until that lands, out-of-acceptance changes from this fix surface as `AcceptanceStatusIgnored` and are simply not applied. | Low | Low | Document the gap in the orchestrator handoff so the GCT item knows to map `Ignored`/`Denied` → `pending-review`/`denied` when the schema lands. Surface here for the orchestrator's attention; do not mutate sibling backlog items from this workshop round. |

## Cross-Initiative Implications (Orchestrator Attention)

- **`execute/gct-pending-ai-provenance-hardening`** (upstream initiative member): this fix removes the bypass but does not introduce the new `state` field on `ProvenanceFile`. The orchestrator should ensure GCT's hardening work maps the existing `AcceptanceInfo.Status` values (`Accepted`/`Ignored`/`Denied`/`BinaryIgnored`) to the new `state` taxonomy (`applied`/`pending-review`/`denied`).
- **`execute/agent-manager-sandbox-auto-apply-defaults`** (sibling, depends on this fix): once this fix lands, the auto-apply work can rely on `evaluateAcceptance` being a true per-file gate regardless of `NoLock`. No further coordination needed.
- **`execute/sandbox-runtime-e2e-verification`** (sibling): Finding 5 behavior #7 ("multiple concurrent sandboxes over the same scope coexist without lock errors, and acceptance still gates each independently") is partially anticipated by this fix's regression tests at the unit + service-integration layer. The e2e item should still own the spawn-surface verification.

## Non-goals / Prohibited Patterns

- **No `state=pending-review` provenance writes.** That schema work is `execute/gct-pending-ai-provenance-hardening`.
- **No `runOutcome` field on provenance.** Same upstream item.
- **No run-end auto-apply wiring.** That is `execute/agent-manager-sandbox-auto-apply-defaults`.
- **No spawn-surface end-to-end verification.** That is `execute/sandbox-runtime-e2e-verification`.
- **No change to `DefaultNoLock`'s value** (currently `true`). Only its doc comment.
- **No `legacy_accept_all` column or any other grandfathering mechanism.** Decision 2A.
- **No removal of `NoLock` itself or the lock-semantics branches** (`pathutil.go:110`, `service.go:437,542`). Locking remains a real, independent concern.
- **No `PROBLEMS.md` entry.** Decision 4A: inline comment + contract doc are sufficient.
- **No production code change to `investigation.go`.** Decision 1A: test-only assertion that investigations don't write.
- **No backwards-compatibility shim, dual-mode flag, or warn-only release.** The prior behavior was a contract violation; the new behavior takes effect immediately on deploy.

## Definition of Done

- [ ] `service.go:1145-1152` bypass branch is removed.
- [ ] Inline comment at the removed bypass site references `AUDITABILITY_CONTRACT.md` Finding 4 M1 (Decision 4A).
- [ ] `evaluateAcceptance` returns identical results for `NoLock=true` and `NoLock=false` given the same acceptance config and file path (verified by Test #3).
- [ ] All seven tests in the Testing Plan pass locally.
- [ ] `config.go` `DefaultNoLock` doc comment explicitly disclaims any influence on acceptance.
- [ ] `AUDITABILITY_CONTRACT.md` line 40 reflects the completed decoupling in past tense.
- [ ] No new test failures in `scenarios/workspace-sandbox/api/internal/sandbox/...`.
- [ ] No new test failures in `scenarios/agent-manager/api/internal/orchestration/...` (investigation flow regression check).
- [ ] `golangci-lint run` and `gofumpt -l` clean for both scenarios.
- [ ] `vrooli scenario restart workspace-sandbox` and `vrooli scenario restart agent-manager` succeed and the stack is healthy.
- [ ] No production code change to `investigation.go` (Decision 1A) — only a new test.
- [ ] No `legacy_accept_all` column or other grandfathering mechanism added (Decision 2A).
- [ ] No spawn-surface e2e test added (deferred to `execute/sandbox-runtime-e2e-verification`).

## Open Questions (resolved)

All workshop decisions resolved in round 1. See **Contract Decisions** above.
