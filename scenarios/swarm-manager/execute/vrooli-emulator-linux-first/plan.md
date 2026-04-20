# Implementation Plan — Retire `execute/vrooli-emulator-linux-first` Umbrella Item

## 1. Purpose

Execute the user's explicit retirement decision from workshop rounds 1 and 2. This plan is a focused retirement-and-cleanup plan: rehome orphan deliverables into existing sibling items, rewire the single dependent edge (`execute/adopt-vrooli-emulator-in-deployment-flows`) onto the actual Phase 1 gating siblings, and archive this item so it no longer appears in active backlog work — without losing the Phase 1 readiness signal for the `emulator-platform` initiative.

## 2. Required Reading

```bash
# Swarm Manager data-model and CLI context
prompt-manager skill read swarm-manager-backlog-tools implementation-plan-authoring
```

Repo-specific context:
- `swarm-manager initiatives get --name emulator-platform` — current rollup (13 items, 2 completed, 11 pending)
- `swarm-manager backlog get --kind execute --name adopt-vrooli-emulator-in-deployment-flows` — the **only** dependent edge; depends_on currently includes this umbrella and must be rewired before closure
- `swarm-manager backlog get --kind chore --name vrooli-emulator-documentation` — sibling that will absorb PRD + runbook + operational-baseline docs per d1
- `swarm-manager backlog get --kind execute --name emulator-acceptance-tests-phase-1` — sibling that will absorb integration smoke + distro matrix per d1
- `workshop/round-001.json` + `workshop/round-002.json` — the settled retirement and orphan-rehoming decisions this plan executes
- `swarm-manager backlog get --kind research --name emulator-extraction-and-service-plan` — original decomposition into 9 sibling items that made this umbrella redundant

## 3. Greenfield Declaration

This is a **greenfield retirement** item — not a code refactor, not a compatibility migration. It performs zero edits inside `scenarios/vrooli-emulator/` or any other scenario: every code-facing deliverable is owned by a sibling item, and those siblings are themselves greenfield (no compatibility shims, no dual-run of legacy livedesktop per research Finding 15). The only mutations this item performs are on swarm-manager data (one item's `depends_on`, this item's `status`), and on two sibling `plan.md` files to absorb rehomed orphan scope. No deprecated code paths are preserved; no transitional adapters are introduced.

## 4. Problem Statement

This item was originally scoped as an XL "build the emulator Linux-first" umbrella. The research item `research/emulator-extraction-and-service-plan` subsequently decomposed that work into 9 concrete sibling backlog items (scaffold, acceptance tests, UI, external-url, iframe embed, livedesktop removal, smoketest delegation, docs, remote-backend spike), leaving this umbrella with no unique direct deliverables. Round 1 asked whether to integrate, retire, or collapse — the user unambiguously chose retire (d1=B, reinforced by "Archive this backlog item" on d2–d5). Round 2 resolved the three remaining mechanics:

- **d1=A**: Orphan deliverables (operational baseline defaults, integration smoke harness, PRD fill-out, runbook) are split across existing siblings — PRD + runbook + baseline-values-as-docs → `chore/vrooli-emulator-documentation`; integration smoke + distro matrix → `execute/emulator-acceptance-tests-phase-1`. No new sibling created.
- **d2=A**: The dependent edge on `execute/adopt-vrooli-emulator-in-deployment-flows` is rewired comprehensively — five Phase 1 siblings replace this umbrella in its `depends_on`.
- **d3=A**: This item is closed via `status=archived` (not deletion), preserving the retirement rationale and round history as a historical record.

Retiring the umbrella cleanly therefore requires four concrete actions:
1. Update two sibling `plan.md` files to formally absorb the rehomed orphan scope (no silent drops).
2. Rewire `execute/adopt-vrooli-emulator-in-deployment-flows.depends_on` to the comprehensive Phase 1 set.
3. Set this item's `status` to `archived`.
4. Verify the initiative rollup and grep for stale references.

## 5. Scope

### In scope
- Apply backlog mutations in this order: (a) append to the two sibling plan.md files, (b) rewire `adopt-vrooli-emulator-in-deployment-flows.depends_on`, (c) archive this item.
- Verify the resulting data-model state via CLI: initiative rollup, dependent-item depends_on, this item's status, and a grep sweep for stale references.
- Document the settled d1/d2/d3 outcomes in §9 Contract Decisions so the retirement rationale is preserved after archive.

### Out of scope
- Any code, UI, docs, or test work inside `scenarios/vrooli-emulator/` or `scenarios/scenario-to-desktop/` — every emulator deliverable is owned by a sibling item, which the user has explicitly directed should remain separate.
- Reopening the retire decision or any of d1–d3.
- Editing sibling scope beyond the minimum absorption needed for d1=A's rehoming (append-only to the two target siblings' plan.md Scope sections).
- The `archive/` folder on this item — spec mandates it is immutable user-provided context.
- Running `vrooli scenario restart` — see §13 Scenario Restart Note; this item makes zero scenario code changes.

## 6. Current Technical Context

**Initiative `emulator-platform` rollup** (as of 2026-04-18):
- Total: 13 items (1 research, 10 execute, 2 chore, 1 research-spike)
- Completed: 2 (`research/emulator-extraction-and-service-plan`, `execute/scaffold-vrooli-emulator-scenario`)
- Pending: 11
- Full item list confirmed via `swarm-manager initiatives get --name emulator-platform`

**Dependent edges into this umbrella** — exactly one: `execute/adopt-vrooli-emulator-in-deployment-flows.depends_on` currently includes `execute/vrooli-emulator-linux-first`. Its full current `depends_on` is:
```
execute/mobile-operable-deployment-surfaces
execute/qa-deployment-manager-code-quality-20260407
execute/qa-scenario-to-desktop-code-quality-20260407
execute/vrooli-emulator-linux-first         ← to be removed
fix/qa-deployment-manager-tests-20260407
```

**Target `depends_on` after rewire** (d2=A — comprehensive Phase 1 gate):
```
execute/mobile-operable-deployment-surfaces
execute/qa-deployment-manager-code-quality-20260407
execute/qa-scenario-to-desktop-code-quality-20260407
fix/qa-deployment-manager-tests-20260407
execute/emulator-acceptance-tests-phase-1            ← new
execute/vrooli-emulator-external-url-endpoint        ← new
execute/scenario-to-desktop-emulator-iframe-embed    ← new
chore/scenario-to-desktop-remove-livedesktop         ← new
chore/vrooli-emulator-documentation                  ← new
```

**Orphan deliverables rehomed per d1=A:**
- → `chore/vrooli-emulator-documentation`: PRD.md fill-out, operator runbook, operational baseline *values-as-docs* (max concurrent sessions, Xvfb display number range, session TTL, per-session resource ceilings, stale-session janitor config). Documentation chore already scopes operator-facing content; this is an append, not a scope pivot.
- → `execute/emulator-acceptance-tests-phase-1`: Phase 1 integration smoke harness (the scripted end-to-end exercise previously envisioned as `make phase1-ready`), plus per-distro validation matrix. Acceptance tests already scope end-to-end validation; this is an append, not a scope pivot.

**acceptance_allow posture:** The spec's `acceptance_allow` is `scenarios/vrooli-emulator/**, scenarios/scenario-to-desktop/**` — legacy from the former build scope. Since this item makes zero code changes and will be archived, the field is inert; we leave it unchanged to avoid spurious churn. Explicitly called out in §12 Risks for the record.

## 7. Target End State

- `execute/vrooli-emulator-linux-first.status == "archived"` — the item still exists on disk, but no longer appears in active queues; plan.md + workshop rounds preserved as historical record (d3=A).
- `execute/adopt-vrooli-emulator-in-deployment-flows.depends_on` matches the §6 "Target" set exactly — `vrooli-emulator-linux-first` removed, five Phase 1 siblings added.
- `chore/vrooli-emulator-documentation/plan.md` Scope section includes PRD fill-out + operator runbook + operational baseline values-as-docs as owned deliverables.
- `execute/emulator-acceptance-tests-phase-1/plan.md` Scope section includes Phase 1 integration smoke harness + distro validation matrix as owned deliverables.
- `emulator-platform` initiative rollup still shows 13 items (archived counts as historical but remains in the rollup); completed count unchanged at 2.
- No other backlog item's `depends_on`, plan.md, workshop rounds, or spec.json references `execute/vrooli-emulator-linux-first` as an active dependency.

## 8. Implementation Strategy

### Phase A — Rehome orphan scope into siblings (d1=A)

1. Fetch `chore/vrooli-emulator-documentation/plan.md` and append a Scope bullet set capturing:
   - PRD.md fill-out (Overview, Operational Targets, Tech Snapshot, Dependencies, UX/Branding stubs) under `scenarios/vrooli-emulator/PRD.md`
   - Operator runbook at `scenarios/vrooli-emulator/docs/runbook.md`
   - Operational baseline *values-as-docs*: max concurrent sessions, Xvfb display number range, session TTL, per-session resource ceilings, stale-session janitor cadence
   - Cite this plan (`execute/vrooli-emulator-linux-first/plan.md §6`) as the rehoming source for auditability.

   Write via:
   ```bash
   swarm-manager backlog file-upload --kind chore --name vrooli-emulator-documentation --path plan.md --stdin <<'EOF'
   <updated plan.md contents>
   EOF
   ```

2. Fetch `execute/emulator-acceptance-tests-phase-1/plan.md` and append a Scope bullet set capturing:
   - Phase 1 integration smoke harness — a scripted end-to-end exercise that stands up the emulator, creates both a VNC session and a headless session, takes a capture, tails metrics, and tears everything down
   - Per-distro validation matrix (at minimum Ubuntu LTS 22.04 and 24.04; other distros deferred until a consumer requests them)
   - Cite this plan (`execute/vrooli-emulator-linux-first/plan.md §6`) as the rehoming source for auditability.

   Write via the analogous `file-upload` call against that sibling's plan.md.

### Phase B — Rewire the dependent edge (d2=A)

3. Update `execute/adopt-vrooli-emulator-in-deployment-flows.depends_on` to the comprehensive set in §6:
   ```bash
   swarm-manager backlog update --kind execute --name adopt-vrooli-emulator-in-deployment-flows \
     --data '{"depends_on":["execute/mobile-operable-deployment-surfaces","execute/qa-deployment-manager-code-quality-20260407","execute/qa-scenario-to-desktop-code-quality-20260407","fix/qa-deployment-manager-tests-20260407","execute/emulator-acceptance-tests-phase-1","execute/vrooli-emulator-external-url-endpoint","execute/scenario-to-desktop-emulator-iframe-embed","chore/scenario-to-desktop-remove-livedesktop","chore/vrooli-emulator-documentation"]}'
   ```

### Phase C — Archive this item (d3=A)

4. After Phase A and Phase B succeed, flip this item's status:
   ```bash
   swarm-manager backlog update --kind execute --name vrooli-emulator-linux-first \
     --data '{"status":"archived"}'
   ```
   Archive is intentionally the final mutation so the historical plan + workshop record already reflects the rehoming and rewiring steps by the time the item leaves active status.

### Phase D — Verify

5. `swarm-manager initiatives get --name emulator-platform` — confirm this item appears with `status=archived`; confirm total count is 13 and completed count is 2 (archived does not count as completed).
6. `swarm-manager backlog get --kind execute --name adopt-vrooli-emulator-in-deployment-flows` — confirm `depends_on` matches §6 target exactly (9 entries, no `vrooli-emulator-linux-first`).
7. `swarm-manager backlog get --kind chore --name vrooli-emulator-documentation` + `file-get --path plan.md` — confirm Scope section now lists the three rehomed deliverables.
8. `swarm-manager backlog get --kind execute --name emulator-acceptance-tests-phase-1` + `file-get --path plan.md` — confirm Scope section now lists the two rehomed deliverables.
9. Grep sweep across the swarm-manager data root for `vrooli-emulator-linux-first`. Expected matches: only this item's own files, the adopt item's historical workshop rounds (allowed; those are append-only audit trails), and the initiative rollup. Any *active* `depends_on` hit other than this item's own files is a bug.

## 9. Contract Decisions

### Settled (authoritative — do not revisit)
- **Role of this item (round 1 d1)**: retire. User selected d1=B and reinforced via "Archive this backlog item" on d2, d3, d4, d5.
- **Orphan disposition (round 2 d1=A)**: PRD + runbook + operational baseline docs → `chore/vrooli-emulator-documentation`; integration smoke + distro matrix → `execute/emulator-acceptance-tests-phase-1`. No new sibling created.
- **Dependent rewire (round 2 d2=A)**: `execute/adopt-vrooli-emulator-in-deployment-flows.depends_on` gets the comprehensive Phase 1 gate — adds `emulator-acceptance-tests-phase-1`, `vrooli-emulator-external-url-endpoint`, `scenario-to-desktop-emulator-iframe-embed`, `scenario-to-desktop-remove-livedesktop`, `vrooli-emulator-documentation`; removes `vrooli-emulator-linux-first`.
- **Retire mechanism (round 2 d3=A)**: `status=archived` via `swarm-manager backlog update`. Not a delete — history is preserved.

### Prohibited by earlier decisions (do not revisit)
- No dual-run of livedesktop (research Finding 15, still binding on siblings).
- No shared `packages/procmetrics` library (research Finding 12, still binding on siblings).
- No reopening the retire decision.
- No creation of a 14th item under this rehoming (d1 chose A, not B).
- No outright deletion of this item (d3 chose A, not B).

## 10. Testing Plan

All verification is mechanical data-model checking via CLI. There are zero code changes in this item, so there is no unit, integration, or scenario test to run here:

1. **Dependent-item rewiring**: `swarm-manager backlog get --kind execute --name adopt-vrooli-emulator-in-deployment-flows` → `depends_on` matches §6 target exactly (9 entries, no `vrooli-emulator-linux-first`). Automated by the Phase D step 6 command.
2. **This item's archival**: `swarm-manager backlog get --kind execute --name vrooli-emulator-linux-first` → `status == "archived"`. Automated by Phase D step 5 (rollup) cross-checked with a direct `get`.
3. **Initiative rollup**: `swarm-manager initiatives get --name emulator-platform` → total=13, completed=2, this item shown with archived status.
4. **Orphan-rehome evidence**: `swarm-manager backlog file-get --kind chore --name vrooli-emulator-documentation --path plan.md` contains the three absorbed deliverables; `file-get --kind execute --name emulator-acceptance-tests-phase-1 --path plan.md` contains the two absorbed deliverables. Both cite this plan as the rehoming source.
5. **Stale-reference grep sweep**: across the swarm-manager data root, no *active* `depends_on` references `vrooli-emulator-linux-first` outside this item's own files.

All five checks are scripted against CLI output and are idempotent — re-running them after any re-application of Phase A–C produces the same result.

## 11. Rollout/Validation Checklist

- [ ] Phase A step 1: `chore/vrooli-emulator-documentation/plan.md` updated with three rehomed deliverables, citing this plan as source.
- [ ] Phase A step 2: `execute/emulator-acceptance-tests-phase-1/plan.md` updated with two rehomed deliverables, citing this plan as source.
- [ ] Phase B step 3: `adopt-vrooli-emulator-in-deployment-flows.depends_on` rewired to the comprehensive set; `get` output matches §6 target exactly.
- [ ] Phase C step 4: this item's `status` flipped to `archived`; `get` confirms.
- [ ] Phase D step 5: initiative rollup shows total=13, completed=2, this item archived.
- [ ] Phase D steps 7–8: both sibling plan.md files confirmed to own the rehomed scope.
- [ ] Phase D step 9: grep sweep returns zero *active* references to this item outside its own files.

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Orphan operational-baseline / integration-smoke work silently dropped if sibling plan.md appends fail or get reverted → emulator ships without ops readiness | Low | High | d1=A forces explicit rehoming into two specific siblings; Phase D steps 7–8 verify the absorption is actually documented; grep sweep catches any partial-rollback state |
| `adopt-in-deployment-flows` rewired to wrong dep set (typo, missing entry, or outdated sibling name) → adopt item starts too early or stalls indefinitely | Low | Medium | Explicit full depends_on list written verbatim in §6 and §8 step 3; Phase D step 6 does a `get` and compares against §6; any mismatch is visible before archival |
| Stale references to this item in other backlog plans' depends_on | Low | Low | Phase D step 9 grep sweep catches any reference the rewire missed; only allowable matches are this item's own files and the adopt item's historical workshop rounds |
| Initiative rollup loses Phase 1 readiness signal once this umbrella is gone | Very Low | Medium | Initiative rollup already derives readiness from per-item status; archive preserves the item in the rollup (just not in active queues), so signal is intact |
| `acceptance_allow` drift on archived item confuses future readers | Low | Low | Leave unchanged; §6 explicitly flags the mismatch as historical artifact; archive status signals the globs are no longer active constraints |
| User later wants to resurrect integration-milestone framing | Very Low | Low | d3=A (archive, not delete) preserves plan + round history; resurrection is a `status` flip, not a reconstruction from memory |

## 13. Scenario Restart Note

**No `vrooli scenario restart` is needed for this item.** This is a backlog-only retirement: zero code is written or modified in any scenario directory (§5 Out-of-scope). The only mutations are swarm-manager data-model writes (sibling plan.md appends, one `depends_on` update, one `status` update). No running scenario observes these changes, so there is no process state to refresh. The validator check for `vrooli scenario restart` is satisfied by this explicit N/A declaration — running a restart here would be a no-op and a misleading signal that code changed.

For future reference: the actual emulator code changes live in the sibling items (`execute/scaffold-vrooli-emulator-scenario` — already completed, `execute/emulator-acceptance-tests-phase-1`, `execute/vrooli-emulator-standalone-ui`, etc.), and each of those plans will carry its own `vrooli scenario restart vrooli-emulator` cleanup step in its Phase-end checklist.

## 14. Non-goals / Prohibited Patterns

- Do not reopen the retire decision — user was emphatic (5× retire-equivalent responses across rounds 1 and 2).
- Do not edit any child-item scope beyond the minimum append needed to absorb the explicitly-chosen orphan work (d1=A targets only `chore/vrooli-emulator-documentation` and `execute/emulator-acceptance-tests-phase-1`).
- Do not make code changes in `scenarios/vrooli-emulator/` or `scenarios/scenario-to-desktop/` under this item — acceptance_allow aside, this item has zero code scope.
- Do not silently drop orphan deliverables — every orphan has an explicit sibling owner per d1=A.
- Do not touch `archive/` on this item or any sibling.
- Do not delete this item — d3=A requires archive, not deletion.
- Do not mark this item complete without running the §11 checklist end-to-end.

## 15. Definition of Done

1. `chore/vrooli-emulator-documentation/plan.md` Scope section owns PRD fill-out + runbook + operational baseline values-as-docs, citing this plan.
2. `execute/emulator-acceptance-tests-phase-1/plan.md` Scope section owns Phase 1 integration smoke harness + distro validation matrix, citing this plan.
3. `execute/adopt-vrooli-emulator-in-deployment-flows.depends_on` matches §6 target exactly (verified via `swarm-manager backlog get`).
4. `execute/vrooli-emulator-linux-first.status == "archived"` (verified via `swarm-manager backlog get` and initiative rollup).
5. `emulator-platform` initiative rollup reports total=13, completed=2, this item listed as archived.
6. Grep sweep confirms no *active* `depends_on` references to this item outside its own files.
7. No `vrooli scenario restart` was run — this item makes zero scenario code changes (§13).
