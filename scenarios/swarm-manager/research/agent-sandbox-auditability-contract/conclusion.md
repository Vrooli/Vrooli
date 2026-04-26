# Research Conclusion: Canonical Sandbox Auditability Contract for agent-manager

## Research Question

What is the authoritative product and implementation contract for `workspace-sandbox` when used as the default execution path for `agent-manager` coding runs, such that:

1. Every coding run produces durable, per-run provenance correlating repository changes back to the agent-manager run, conversation, cost, and execution context that produced them.
2. The default operator path requires no manual review and no special agent-side awareness of sandbox mode.
3. Acceptance, locking, manual-review, auto-apply, and apply-on-failure are clearly distinct concerns with named defaults.
4. The contract reconciles the existing semantic mismatches in `workspace-sandbox`, `agent-manager`, `cli-core`, `test-genie`, and `git-control-tower` so we can make sandboxing the default in major spawn surfaces without ambiguity.

## Summary

The contract names a single auditability-first mode (`tracking`, with `protected` reserved for future work) whose defaults are `manualReview=false`, `autoApply=true`, `applyOnFailure=true`, `lock=false`, and `networkMode=localhost`. Locking and acceptance are orthogonal: `NoLock` controls only mutual exclusion, while acceptance independently gates apply eligibility. Sandboxes are created **eagerly** at run start so every coding run has a provenance record (even no-op runs). Provenance gains two schema fields: `runOutcome` ∈ {success, failure, cancelled, timeout} on each `ProvenanceRunGroup`, and `state` ∈ {applied, pending-review, denied} on each `ProvenanceFile` so GCT can filter the existing API rather than depending on a new endpoint. When operators opt into `manualReview=true`, apply is deferred until approval in any of the three viewing surfaces (git-control-tower, agent-manager, workspace-sandbox), and the sandbox persists until then. Five mismatches between current code and this contract are enumerated and routed to existing initiative members. Sandboxing flips to default after the validation matrix passes on the agent-manager UI and swarm-manager queue spawn surfaces; cron and direct-CLI surfaces flip in follow-on items.

## Methodology

The research is conducted as a structured contract-definition exercise rather than an empirical study. Sources and methods:

1. **Code reading of the source-of-truth surfaces** for sandbox lifecycle, env injection, apply behavior, and provenance:
   - `scenarios/agent-manager/api/internal/orchestration/run_executor.go` (sandbox env injection, run lifecycle)
   - `scenarios/workspace-sandbox/api/internal/sandbox/{service.go,lifecycle.go,heal.go,pathutil.go}` (overlay, apply, NoLock semantics)
   - `scenarios/workspace-sandbox/api/internal/config/config.go` (PolicyConfig including RequireHumanApproval, DefaultNoLock, TeardownHooks; ExecutionConfig)
   - `scenarios/workspace-sandbox/api/internal/toolregistry/*.go` (tool-level RequiresApproval flags)
   - `packages/cli-core/cliutil/sandbox.go` and `packages/cli-core/cmd/sandbox-resolve/main.go` (consumer-side env resolution)
   - `internal/scenario/scenario.go`, `internal/scenarioexec/subprocess.go`, `internal/cli/vroolicli/runtime.go`, `internal/repocontractcheck/checks.go` (canonical scenario-restart and env-propagation paths in Go — NOT the stale `scripts/lib/scenario/**` and `cli/commands/scenario/**` referenced in the original spec)
   - `scenarios/test-genie/cli/execute/command.go` (sandbox-aware test path)
   - `scenarios/git-control-tower/api/{approved_changes_handler,approved_changes_model}.go` and `scenarios/git-control-tower/ui/src/components/AIProvenanceTab.tsx` (provenance API, AI Changes UI, ProvenanceRunGroup schema)

2. **Cross-surface state mapping**: produce a matrix of how each surface participates in sandbox-aware behavior today (env vars consumed, apply/restart paths, provenance writes, UI reads) so the contract names each touchpoint explicitly.

3. **Known-mismatch reconciliation**: walk the four mismatches called out in the spec (noLock-vs-acceptance conflation, RequiresApproval-without-apply, undocumented end-to-end contract, undefined provenance lifecycle) plus the fifth uncovered during round 1 (stale acceptance-allow path references) and decide for each: keep current, change behavior, or rename/relabel.

4. **Workshop-driven decisions**: lock contract semantics through two workshop rounds. Round 1 locked the high-level shape (mode taxonomy, network default, failure handling, provenance lifecycle, acceptance-overflow handling, stale-path resolution, mirror location). Round 2 locked the finer schema and policy questions (provenance outcome shape, apply timing under `manualReview`, sandbox creation timing, out-of-acceptance review-queue ownership, validation-matrix surface scope).

5. **Validation matrix design**: define the behaviors that must be observable end-to-end before sandboxing flips to default in major spawn surfaces.

No experiments or benchmarks are required; the deliverable is a written contract plus a reconciliation checklist.

## Findings

### Finding 1: Locked-in default contract

The auditability-first contract names a single named mode with the following defaults. Each default is anchored to the workshop decision that locked it.

| Lever | Default | Source |
|---|---|---|
| `mode` | `tracking` (with `protected` reserved as a future value that errors if requested before its implementation lands) | round-1 d1=A |
| `manualReview` | `false` (opt-in only). When `true`, apply is **deferred** until operator approval (see Finding 2) | initiative context + round-2 d2=B |
| `autoApply` | `true` | initiative context + spec |
| `applyOnFailure` | `true` (identical apply behavior regardless of run outcome; outcome captured in provenance metadata) | round-1 d3=A |
| `lock` | `false` (locking and acceptance are orthogonal; multiple concurrent sandboxes over the same scope are allowed) | initiative context + spec |
| `networkMode` | `localhost` (supports `none|localhost|full`) | round-1 d2=A |
| `acceptance` | governs apply eligibility only; in-acceptance changes auto-apply, out-of-acceptance changes are retained in the sandbox and surfaced via a Git Control Tower review-queue entry the operator can manually approve | round-1 d5=B |
| `provenanceLifecycle` | pending provenance auto-promotes to committed when GCT detects a commit whose changed files overlap a pending record | round-1 d4=A |
| `sandboxCreation` | eager — sandbox is created at run start regardless of whether the agent writes anything; empty provenance entries are valid and expected for no-op runs | round-2 d3=A |
| Agent-side awareness | none — agent prompts and behavior do not vary based on sandbox mode | initiative context + spec |

### Finding 2: Configuration surface and apply-timing state machine

The configuration surface is exposed at three layers, each with a single source of truth.

- **agent-manager run input** (per-run): `mode`, `manualReview`, `autoApply`, `applyOnFailure`, `lock`, `networkMode`, `acceptanceAllow`, `acceptanceDeny`. Defaults from Finding 1; explicit values override. None of these are passed through to the agent prompt.
- **workspace-sandbox `PolicyConfig`** (operator-tunable, deployment-wide): aligns with the per-run defaults above. `DefaultNoLock` (currently `true`) is preserved but the contract decouples it from acceptance evaluation (see Finding 4 M1).
- **`acceptanceAllow` / `acceptanceDeny`** (per-run, declarative): glob patterns. Used in two distinct ways under the contract: (a) gating which changes are eligible for auto-apply, and (b) post-execution review for flagging deviations. These two uses share the same field but are evaluated independently.

**Apply-timing state machine** (round-2 d2=B):

| Mode | Run end | Sandbox lifetime | Apply trigger |
|---|---|---|---|
| `manualReview=false` (default) | Auto-apply runs immediately for in-acceptance changes; out-of-acceptance changes remain in the sandbox as `pending-review` provenance | Sandbox can be torn down once apply completes (and after any pending-review entries are persisted as provenance — the file content lives in provenance, not in the sandbox) | Run end, automatic |
| `manualReview=true` (opt-in) | No apply at run end; all changes are persisted as `pending-review` provenance | Sandbox **persists** until the operator approves or denies | Operator approval action |

The operator can approve a pending-review run from any of three surfaces, all of which surface the same provenance-by-run data: **git-control-tower** (AI Changes tab), **agent-manager** (run-detail diff view), or **workspace-sandbox** (sandbox-detail diff view). All three call the same approve/deny endpoint on workspace-sandbox; the originating surface is recorded on the resulting provenance state transition for audit purposes.

**Provenance schema additions** (round-2 d1=A and d4=B):

| Field | Where it lives | Values | Purpose |
|---|---|---|---|
| `runOutcome` | `ProvenanceRunGroup` (run-level) | `success`, `failure`, `cancelled`, `timeout` | Lets GCT render failed-run provenance differently from successful-run provenance; honors round-1 d3=A's "identical apply, distinct rendering" framing |
| `state` | `ProvenanceFile` (per-file) | `applied`, `pending-review`, `denied` | Replaces the need for a separate out-of-acceptance endpoint; GCT filters the existing provenance API by `state=pending-review` to render the review queue (round-1 d5=B). Per-file granularity lets a single run mix applied (in-acceptance) and pending-review (out-of-acceptance) files without splitting the run group |

Per-file `state` was chosen over per-run state because a single run frequently produces a mix of in-acceptance and out-of-acceptance edits; keeping the run group whole preserves the auditability rollup while still letting acceptance gate individual files.

### Finding 3: Source-of-truth interaction matrix

| Surface | Owns | Reads from | Writes to |
|---|---|---|---|
| `agent-manager` run executor | Run lifecycle, eager sandbox creation at run start, sandbox env injection (`VROOLI_SANDBOX_ID`, `VROOLI_SANDBOX_MERGED`, `VROOLI_SANDBOX_SCOPE`), per-run config defaults | workspace-sandbox CreateSandbox, workspace-sandbox Apply at run end (when `manualReview=false`) | provenance metadata (run_id, conversation_id, cost, runOutcome) attached to apply call |
| `agent-manager` UI | Run-detail diff view; can be one of the three approval surfaces for `pending-review` provenance | GCT provenance API (filtered to current run) | Approve/deny calls to workspace-sandbox |
| `workspace-sandbox` service | Overlay creation, file mutation tracking, acceptance evaluation, apply to canonical repo, teardown hooks, **persistence beyond run end when `manualReview=true`** | run env vars (none — receives explicit args via API) | per-run provenance records (sandbox_id, run_id, files, change_type, applied_at, runOutcome, per-file state) |
| `workspace-sandbox` UI | Sandbox-detail diff view; can be one of the three approval surfaces for `pending-review` provenance | GCT provenance API (or workspace-sandbox local view) | Approve/deny endpoint (local) |
| `workspace-sandbox` toolregistry | Tool-level `RequiresApproval` flags (currently used to gate canonical-repo-modifying tools that bypass the sandbox, e.g. direct git commit) | — | — |
| `internal/scenario` + `internal/scenarioexec` + `internal/cli/vroolicli` (Go-based scenario-restart and lifecycle) | Sandbox-aware scenario restart, sandbox-aware test execution, env propagation to subprocesses, scope-narrowed redirect (only scenarios within `VROOLI_SANDBOX_SCOPE` use the merged path) | `VROOLI_SANDBOX_*` env vars set by agent-manager | — |
| `packages/cli-core/cliutil/sandbox.go` + `packages/cli-core/cmd/sandbox-resolve` | Path resolution helper for arbitrary CLIs that need to resolve paths against the sandbox | `VROOLI_SANDBOX_*` env vars | — |
| `test-genie` CLI | Sandbox-aware test execution paths | sandbox env vars | — |
| `workspace-sandbox` `TeardownHooks` | Triggers `vrooli scenario heal-from-sandbox` on sandbox teardown to restart any scenarios still running from the merged path | — | invokes Go-based `vrooli scenario heal-from-sandbox` (not the stale `cli/commands/scenario/modules/heal.sh`) |
| `git-control-tower` API | Provenance-by-run query (`ProvenanceRunGroup` by `runId`), state-filtered query (`state=pending-review` for review queue), approved-changes preview, commit-time linkage | workspace-sandbox provenance records | linkage records (pending → committed); does NOT write `state` itself — workspace-sandbox owns state transitions |
| `git-control-tower` UI (`AIProvenanceTab.tsx`) | Surfaces per-run provenance in AI Changes tab; renders the review queue by filtering on `state=pending-review`; can be one of the three approval surfaces | GCT provenance API | Approve/deny calls (proxied to workspace-sandbox) |

### Finding 4: Reconciliation of current mismatches

| Mismatch | Current state | Resolution under the contract | Owning follow-on item |
|---|---|---|---|
| **M1: noLock conflated with acceptance bypass** | `workspace-sandbox/api/internal/sandbox/service.go:1146-1147` explicitly: "When noLock is true, all files are automatically accepted (no acceptance rules apply)". | Decouple: `NoLock` controls only mutual exclusion. Acceptance evaluation runs independently regardless of `NoLock`. | `fix/workspace-sandbox-lock-and-acceptance-semantics` |
| **M2: `RequiresApproval=false` runs can complete without applying sandbox changes** | Tool-level `RequiresApproval` flags exist in `toolregistry/*.go` but the run-end apply-by-default behavior is not wired through agent-manager's run executor. | Run-end apply becomes the default (`autoApply=true`). The tool-level `RequiresApproval` flag is preserved for canonical-repo-modifying tools that bypass the sandbox (e.g., direct git commit), not for normal sandboxed file edits. | `execute/agent-manager-sandbox-auto-apply-defaults` |
| **M3: end-to-end contract not documented in one place** | Sandbox env injection, downstream restart, test flow, and apply behavior live across 6+ packages with no canonical doc. | This conclusion + a mirror at `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` (round-1 d7=A). | this research's `Update document` actions |
| **M4: pending-vs-committed provenance lifecycle undefined; no run outcome or per-file state in schema** | `ProvenanceRunGroup` schema exists with `runId`, `sandboxId`, files, `latestAppliedAt` — but no rule for pending-to-committed promotion, no `runOutcome`, no per-file `state`. | Auto-promote on commit detection: GCT links a commit to a pending provenance record when their changed-file sets overlap (round-1 d4=A). Add `runOutcome` to `ProvenanceRunGroup` and `state` to `ProvenanceFile` (round-2 d1=A, d4=B). | `execute/gct-pending-ai-provenance-hardening` (downstream initiative member; the schema additions belong here, not in agent-manager) |
| **M5: stale acceptance_allow path references** (uncovered in round 1) | `spec.json` and the initiative context reference `scripts/lib/scenario/**` and `cli/commands/scenario/**`. Neither exists. The actual code is in Go at `internal/scenario/`, `internal/scenarioexec/`, `internal/cli/vroolicli/`, `internal/repocontractcheck/`. | Update this item's `acceptance_allow` to reference the actual Go paths; update the initiative context to reflect the migration. | this research's `Update backlog item` action on itself + `Update document` on the initiative context |

### Finding 5: Validation matrix

The behaviors below must be observable end-to-end before sandboxing flips to default. **Surface scope** (round-2 d5=B): all eight behaviors must pass on the **agent-manager UI** and **swarm-manager queue** spawn surfaces before the default flips. The cron and direct-CLI spawn surfaces flip in follow-on items, not as a precondition for the rollout.

1. Every sandboxed agent-manager run produces a provenance record correlating sandbox → run → conversation → cost. Eager creation means even no-op runs leave an empty provenance entry.
2. Failed runs that produced changes still write provenance and still auto-apply (subject to acceptance), with `runOutcome` captured on the provenance record.
3. Restart and test flows operate against the merged sandbox path when `VROOLI_SANDBOX_*` env vars are present, and against the canonical repo otherwise.
4. `acceptanceDeny` violations never reach the canonical repo, even with `autoApply=true`.
5. Out-of-`acceptanceAllow` changes are retained as `state=pending-review` provenance and visible in GCT's AI Changes review queue (and in the agent-manager and workspace-sandbox diff views).
6. GCT auto-links a commit to a pending provenance record when their changed-file sets overlap.
7. Multiple concurrent sandboxes over the same scope coexist without lock errors (per `lock=false` default), and acceptance still gates each independently.
8. `vrooli scenario heal-from-sandbox` (Go) restarts any scenarios still running from the merged path on sandbox teardown.
9. `manualReview=true` defers apply until operator approval; the sandbox persists across run end; approving from any of the three surfaces (GCT, agent-manager, workspace-sandbox) produces the same applied state and records the originating surface on the state transition.

### Finding 6: Rollout sequencing recommendation

Aligned with the parent initiative's existing member ordering:

1. **Contract authored** (this item) — locks defaults, terminology, state transitions, schema additions.
2. **Semantics fixes** (`fix/workspace-sandbox-lock-and-acceptance-semantics`) — decouple `NoLock` from acceptance.
3. **Auto-apply defaults wired** (`execute/agent-manager-sandbox-auto-apply-defaults`) — run-end apply default-on with eager sandbox creation and `runOutcome` recorded.
4. **GCT provenance schema hardened** (`execute/gct-pending-ai-provenance-hardening`, upstream initiative) — adds `runOutcome` to `ProvenanceRunGroup` and `state` to `ProvenanceFile`; implements `state=pending-review` filtering. This is a hard prerequisite for items 5 and the rollout.
5. **End-to-end verification** (`execute/sandbox-runtime-e2e-verification`) — adopts Finding 5 as acceptance criteria across the agent-manager UI and swarm-manager queue surfaces.
6. **Default rollout** (`execute/agent-manager-default-sandboxing-rollout`) — flip the default in agent-manager UI and swarm-manager queue spawn surfaces, gated on Finding 5 passing on those surfaces. Cron and direct-CLI surface flips are scoped as separate follow-on items, not blockers for the initial rollout.

Protected-mode work (containment, runtime guardrails, git read-only enforcement) sequences after this initiative completes and lives in the upstream `protected-agent-sandboxing` initiative.

## Limitations

- **Stale path references in original `acceptance_allow`.** Confirmed in round 1 and resolved in Finding 4 (M5): the canonical scenario-restart code is Go in `internal/`, not bash in `scripts/lib/scenario/` or `cli/commands/scenario/`. The initiative context document references the same stale paths and needs an update.
- **Protected-mode design is explicitly downstream.** This research locks contract decisions that protected mode must honor, but it does not design protected-mode containment, runtime guardrails, or git restriction enforcement. Those belong to the upstream `protected-agent-sandboxing` initiative.
- **Undo/revert workflow is acknowledged but out of scope.** Operators will need it to feel safe with auto-apply-by-default, but it is a separate first-class workflow tracked under the downstream `run-level-undo-and-revert` initiative.
- **Empirical validation is deferred.** This research produces the contract and the validation matrix. The actual end-to-end verification runs in `execute/sandbox-runtime-e2e-verification`.
- **Cron and direct-CLI spawn surfaces are out of scope for the initial default flip.** Round-2 d5=B scopes the validation gate to the agent-manager UI and swarm-manager queue. Cron and CLI flip independently in follow-on items, which leaves a known (but accepted) gap in coverage at the moment the default flips.
- **Sandbox-persistence-beyond-run-end (manualReview=true) introduces a new lifecycle responsibility.** Today's sandbox is implicitly tied to a run; round-2 d2=B requires it to outlive the run when `manualReview=true`. The exact garbage-collection rule for abandoned pending-review sandboxes (e.g., TTL, operator action timeout) is left to the implementing item to specify, not pre-decided here. This is the most consequential follow-on detail.

## Actions

The actions below are the concrete follow-ups that close out this research and propagate the locked contract into the rest of the initiative. All `Update backlog item` actions name existing initiative members; no new members are required.

### Action 1: Update backlog item — Tighten fix/workspace-sandbox-lock-and-acceptance-semantics
- **Kind**: fix
- **Name**: workspace-sandbox-lock-and-acceptance-semantics
- **Changes**:
  - description: encode the resolution from Finding 4 M1 verbatim (decouple `NoLock` from acceptance evaluation; remove the conditional in `service.go:1146-1147` so acceptance rules apply regardless of `NoLock`).
  - acceptance_allow: ensure it covers `scenarios/workspace-sandbox/api/internal/sandbox/**` and the assumption tests under that path.
- **Reason**: Finding 4 M1 pins the exact mismatch and the exact code site; the existing item's description is more abstract than the locked contract now allows.

### Action 2: Update backlog item — Tighten execute/agent-manager-sandbox-auto-apply-defaults
- **Kind**: execute
- **Name**: agent-manager-sandbox-auto-apply-defaults
- **Changes**:
  - description: encode Finding 1's defaults table (including `sandboxCreation=eager`), Finding 2's per-run config surface, and the run-end apply timing for `manualReview=false`. Require provenance writes to populate `runOutcome` and per-file `state` (depends on Action 5's GCT schema work landing first or in parallel). Apply behavior is identical regardless of run outcome (round-1 d3=A).
- **Reason**: Finding 1 and Finding 2 provide a precise, contract-grade specification of the defaults this item must encode; round-2 d3 and d2 add the eager-creation and apply-timing requirements that were previously open.

### Action 3: Update backlog item — Tighten execute/sandbox-runtime-e2e-verification
- **Kind**: execute
- **Name**: sandbox-runtime-e2e-verification
- **Changes**:
  - description: adopt Finding 5's nine validation behaviors as acceptance criteria; restrict the gating surface scope to the agent-manager UI and swarm-manager queue (round-2 d5=B).
- **Reason**: Finding 5 defines the must-pass behavior set the rollout depends on; round-2 d5 narrows the surface scope.

### Action 4: Update backlog item — Tighten execute/agent-manager-default-sandboxing-rollout
- **Kind**: execute
- **Name**: agent-manager-default-sandboxing-rollout
- **Changes**:
  - description: encode Finding 6's sequencing and gate the rollout on Finding 5 passing on the agent-manager UI and swarm-manager queue surfaces only; explicitly note that cron and direct-CLI surface flips are out of scope for this item and tracked separately.
- **Reason**: Finding 6 finalizes the sequencing; round-2 d5=B narrows the rollout surface.

### Action 5: Update backlog item — Tighten execute/gct-pending-ai-provenance-hardening (upstream initiative member)
- **Kind**: execute
- **Name**: gct-pending-ai-provenance-hardening
- **Initiative**: git-control-tower-ai-provenance (upstream)
- **Changes**:
  - description: require the schema additions locked by round-2 d1=A and d4=B — add `runOutcome` ∈ {success, failure, cancelled, timeout} to `ProvenanceRunGroup`; add `state` ∈ {applied, pending-review, denied} to `ProvenanceFile`; require `state=pending-review` filtering on the provenance query API to power the AI Changes review queue. Workspace-sandbox owns state transitions; GCT only reads them.
- **Reason**: This is the load-bearing schema work for Findings 2 and 4 (M4). It lives in the upstream initiative, not this one, but this contract pins the exact shape it must produce. If this item already exists, this is an `Update`; if it does not, the upstream initiative needs a `Create` action — verified in initiative context output that `gct-pending-ai-provenance-hardening` is referenced as a dependency of `execute/agent-manager-default-sandboxing-rollout`, so the item exists and this is an Update.

### Action 6: Update backlog item — Correct this research item's acceptance_allow [COMPLETED during finalize]
- **Kind**: research
- **Name**: agent-sandbox-auditability-contract
- **Changes**:
  - acceptance_allow: replaced `scripts/lib/scenario/**` and `cli/commands/scenario/**` with `internal/scenario/**`, `internal/scenarioexec/**`, `internal/cli/vroolicli/**`, `internal/repocontractcheck/**`. All other globs preserved.
- **Reason**: Finding 4 M5 confirms the stale paths and identifies the canonical Go locations. This was executed inline during finalize because the workspace-sandbox finalize-upload validator blocks uploads when acceptance globs reference non-existent paths; the contract had to self-heal before the finalize round could be persisted.

### Action 7: Update document — Mirror the contract into the workspace-sandbox docs
- **File**: scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md
- **Change**: Create a new file mirroring this conclusion's Findings 1–6 as a discoverable contract document co-located with the implementation. This satisfies round-1 d7=A (contract lives at the workspace-sandbox docs surface).

### Action 8: Update document — Refresh the initiative context
- **File**: scenarios/swarm-manager/initiatives/agent-sandbox-audit-foundation/context/2026-04-09-sandbox-audit-foundation.md
- **Change**: Append a short "Locked Contract" section pointing to this conclusion and the AUDITABILITY_CONTRACT.md mirror; update the stale path references (`scripts/lib/scenario/runner.sh`, `cli/commands/scenario/modules/heal.sh`) to their actual Go locations.

### Initiative-scope review
No `Delete backlog item` or `Update initiative` actions are required:
- All five existing initiative members remain valid and load-bearing under the locked contract; the contract sharpens their descriptions but does not invalidate any of them.
- The initiative's `depends_on` (`protected-agent-sandboxing`, `git-control-tower-ai-provenance`) is correct: `gct-pending-ai-provenance-hardening` (Action 5) lives upstream and is a true prerequisite, and protected-mode work remains downstream of this initiative.
- Priority remains appropriate as locked.
