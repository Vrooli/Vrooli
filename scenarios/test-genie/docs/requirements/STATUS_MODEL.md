# Requirement & Operational-Target Status Model

This document explains the status numbers Test Genie prints in the
**REQUIREMENTS & OPERATIONAL TARGETS** block at the end of every `test-genie
execute` run, and how those numbers are derived from your tests. It is the
companion to [IMPROVING_COVERAGE.md](IMPROVING_COVERAGE.md), which explains how
to move the numbers up.

## The two things being counted

Every scenario can declare what it is supposed to do in two linked places:

- **PRD operational targets (`OT-P{0,1,2}-NNN`)** — the product-level promises
  in `PRD.md`, each a checkbox. `P0` is critical, `P1` major, `P2` minor.
- **Technical requirements** — the concrete, testable units under
  `requirements/` (`index.json` plus per-domain `module.json` files). Each
  requirement links *up* to an operational target via its `prd_ref` field and
  *down* to tests via its `validation[]` entries.

```
PRD.md
  └─ OT-P0-001  "Users can sign in"      [ ] / [x]   ← checkbox flips automatically
        ▲ prd_ref
        │
   requirements/02-auth/module.json
      ├─ REQ-AUTH-001  status: complete     ─┐
      ├─ REQ-AUTH-002  status: in_progress   ├─ roll up to the OT
      └─ REQ-AUTH-003  status: complete     ─┘
            ▲ validation[]
            │
        tests tagged [REQ:REQ-AUTH-001]  → pass / fail
```

## The three status layers

Status is derived bottom-up across three layers (see
`api/internal/requirements/types/status.go`):

1. **Live status** (from this run's evidence): `passed`, `failed`, `skipped`,
   `not_run`, `unknown`. When several tests cover one validation the rollup
   takes the worst: `failed > skipped > passed > not_run > unknown`.
2. **Validation status** (derived from live): a passing test makes a validation
   `implemented`; a failing test makes it `failing`; skipped/not-run →
   `planned`.
3. **Declared requirement status**: `pending` → `planned` → `in_progress` →
   `complete` (plus `not_implemented`). Test Genie adjusts this from the
   validations:
   - A requirement becomes **`complete`** only when **every** validation passes.
   - If **any** validation is **failing**, a `complete` requirement is demoted
     back to **`in_progress`** — this is the *regression* the report flags.

## How an operational-target checkbox flips

An operational target is **complete only when every requirement that links to
it (`prd_ref`) is itself `complete`**. When that holds, Test Genie checks the
box in `PRD.md` (`- [x] OT-P0-001 …`); if a linked requirement later regresses,
the box is unchecked again. The PRD is backed up (timestamped) under
`coverage/requirements-sync/prd-backups/` before each rewrite.

This is exactly the `OTComplete / OTTotal` number in the report.

## Why a partial run shows "Not updated this run"

Syncing requirement status writes to disk, so it only runs when the evidence is
trustworthy — i.e. when the **full required phase set ran**. A targeted run
(`--phase unit`, or any run where a required phase was skipped or missing) does
**not** re-derive status, because it only saw part of the picture and could
wrongly demote requirements whose tests simply didn't run this time.

When that happens the report still shows the **last persisted** counts, clearly
flagged:

```
REQUIREMENTS & OPERATIONAL TARGETS:
  • Operational targets: 3/7 complete  (P0 2/2 · P1 1/3 · P2 0/2)
  • Requirements: 18/25 complete  (in_progress 4 · planned 2 · pending 1)
  ⚠ Not updated this run — required phases skipped: integration.
    Counts are from the last full sync (2026-05-29).
    To refresh: test-genie execute <scenario>
```

The counts are real (read from `requirements/**/module.json` and the sync
metadata), just not recomputed this run. Run the full suite to refresh them.

## What a full run adds

When the full suite runs, sync executes and the report additionally shows the
**changes this run** — promotions (`… → complete`) and, prominently,
**regressions** (`complete → in_progress`), because a shipped capability
silently breaking is the single most important thing to surface:

```
REQUIREMENTS & OPERATIONAL TARGETS:
  • Operational targets: 4/7 complete  (P0 2/2 · P1 2/3 · P2 0/2)
  • Requirements: 20/25 complete  (in_progress 3 · planned 2)
  ▲ 2 requirement(s) now complete:
      ▲ REQ-AUTH-002 (OT-P0-001) in_progress → complete
  ⛔ REQUIREMENTS & OPERATIONAL TARGETS: 1 regressed this run
  ▼ 1 requirement(s) regressed:
      ▼ REQ-BILLING-004 (OT-P1-002) complete → in_progress
```

## Where this lives in the code

- Status model & derivation: `api/internal/requirements/types/status.go`
- OT extraction & checkbox flips: `api/internal/requirements/sync/syncer.go`
  (`OperationalTargetSummary`, `desiredOperationalTargetCheckboxes`)
- Sync orchestration & gating: `api/internal/orchestrator/suite_execution.go`
  (`syncRequirementsIfNeeded`) and `requirements_decision.go`
- Report rendering: `cli/execute/report/printer.go`
  (`printRequirementsSummary`)
