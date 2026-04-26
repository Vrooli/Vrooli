# Research Conclusion: Canonical Sandbox Auditability Contract

## Research Question

What is the authoritative product and implementation contract for agent-manager sandboxing that makes sandbox the default path for agent-created code changes, with auditability (not security) as the primary value proposition?

**Success criteria:** A written contract with precise defaults, terminology, state transitions, configuration surface, interaction matrix, validation matrix, and rollout recommendation — sufficient for implementation without ambiguity.

## Summary

Sandbox should become the default execution path for agent-manager coding runs because every run produces a per-run, file-level audit trail that Git Control Tower can correlate to conversation, cost, and outcome. The two blocking semantic mismatches (lock/acceptance conflation, approval-gated apply) must be fixed first; auto-apply (including on failure) and provenance enrichment then unlock the auditability value prop. Protected-mode hardening (network, read-only git, resource limits) follows on the same contract — it adds runtime guarantees without changing the auditability shape.

## Constraints & Scope

- **In scope:** agent-manager, workspace-sandbox, cli-core, test-genie, scenario runner, Git Control Tower — their interactions during sandboxed coding runs
- **Out of scope:** Non-coding agent runs (chat-only, research), hardware/deployment topology, UI design beyond provenance display
- **Key constraint:** The contract must make sandbox transparent to the coding agent — no prompt or behavior changes based on sandbox mode

## Methodology

- Direct code reading of the eight reference files in the spec (round 1)
- Cross-referencing config, service, and orchestration layers across agent-manager, workspace-sandbox, cli-core, test-genie, scenario runner, and Git Control Tower
- Synthesis of findings against the spec's contract decisions to identify mismatches
- Round 2: refining the configuration surface to implementation-ready field specs (per d1=A), specifying default behavior on failure (per d2=A), and listing required provenance categories (per d3=A); adding validation matrix, source-of-truth matrix, and risks

## Findings

### F1: Locking and Acceptance Are Currently Conflated (Critical Mismatch)

**Source:** `workspace-sandbox/api/internal/sandbox/service.go`, `evaluateAcceptance()` (lines 1146-1151)

When `NoLock=true` (the current default), the service bypasses acceptance rules entirely — all files are auto-accepted. The code comment reads: "all files are automatically accepted (no acceptance rules apply)."

This directly contradicts the spec requirement that locking and acceptance are independent concerns. The intended contract:
- **Locking** = mutual exclusion on scope paths (can multiple sandboxes target the same area?)
- **Acceptance** = file-level filtering on approval (which changes are eligible to auto-apply?)

Today they are coupled: `noLock` implies `noAcceptance`. Fixing this is the highest-priority semantic change and is owned by sibling item `fix/workspace-sandbox-lock-and-acceptance-semantics`.

### F2: Auto-Apply Defaults Block the Auditability Path (Critical Mismatch)

**Source:** `agent-manager/api/internal/orchestration/run_executor.go` (lines 1119-1168, 1402-1496)

Current defaults:
- `RequiresApproval = true` (line 1123)
- Default auto-approval strategy: `AutoApproveIfEmpty` — only auto-approves if the sandbox diff is empty

Every sandboxed run that produces actual code changes lands in `NeedsReview`. The spec wants `RequiresApproval=false` by default, auto-apply at run end (including on failure for accepted changes), and acceptance rules — not manual review — gating what gets applied. Owned by sibling item `execute/agent-manager-sandbox-auto-apply-defaults`.

### F3: Provenance Pipeline Exists But Lacks Execution Context

**Source:** `git-control-tower/api/approved_changes_handler.go`, `AIProvenanceTab.tsx`

Provenance grouping works (changes grouped by `AgentManagerRunID`, `SandboxID`, `SandboxOwner`), but execution context is missing: no task description, conversation summary, cost, duration, failure reason, or drill-through. For auditability to be the primary value prop, provenance must answer: "what agent, doing what task, at what cost, produced these changes and why?"

### F4: Sandbox State Machine Is Sound

The state machine `Creating → Active ↔ Stopped → Approved/Rejected` with partial approval, `BaseCommitHash` conflict detection, rebase, audit trail, and pre-teardown hooks is a solid foundation. The contract should formalize it as-is and layer new defaults on top.

### F5: Environment Injection and Path Resolution Achieve Agent Transparency

`VROOLI_SANDBOX_ID`, `VROOLI_SANDBOX_MERGED`, `VROOLI_SANDBOX_SCOPE` plus `cli-core/cliutil/sandbox.go DetectSandbox()/ResolveScenarioPath()` make the sandbox invisible to the coding agent. Test-genie and scenario runner both use this. The contract canonicalizes these three env vars as the detection interface.

### F6: Two-Mode Model Needs Formal Definition

- **Tracking mode:** Overlayfs sandbox for change isolation and provenance. No runtime restrictions beyond the overlay. Near-term default.
- **Protected mode:** Adds network restrictions, read-only git, resource limits. Future hardening layer over the same auditability contract.

`NetworkMode` and `ExecutionConfig` exist in config but are not enforced as a coherent "mode."

### F7: Network Mode Configuration Exists But Enforcement Is Unclear

Modes `none`/`localhost`/`full` are required because scenario CLIs depend on localhost APIs. The enforcement mechanism (iptables, namespaces, bwrap) is not visible in the service layer.

### F8: Scenario Runner Scope Validation Has a Silent Fallback

**Source:** `scripts/lib/scenario/runner.sh` (lines 253-278)

When sandbox scope is too narrow for lifecycle operations (e.g., `scenarios/my-app/ui` instead of `scenarios/my-app`), the runner silently falls back to the real repo path — agent changes can become invisible during restart/test with no warning. This must warn loudly, not fall back silently.

### F9: Configuration Surface (Implementation-Ready)

Per d1=A, the contract specifies these fields. They live on the agent-manager run config and are forwarded to workspace-sandbox at sandbox creation:

| Field | Type | Default | Owner | Description |
|-------|------|---------|-------|-------------|
| `manualReview` | bool | `false` | agent-manager | When true, run completes into `NeedsReview` regardless of acceptance outcome. Operator must approve/reject. Replaces today's `RequiresApproval` default. |
| `autoApply` | bool | `true` | agent-manager | When true, accepted changes apply at run end without manual review. Replaces `AutoApproveIfEmpty`. Mutually exclusive guidance: `manualReview=true` overrides `autoApply`. |
| `applyOnFailure` | bool | `true` | agent-manager | When true, accepted changes auto-apply even when the run terminates in failure. Per d2=A, default ON to preserve auditability coverage. Operators opt out per spawn surface for critical paths. |
| `lock` | enum `none\|scope\|exclusive` | `none` | workspace-sandbox | Locking strategy. `none` allows concurrent sandboxes on overlapping scope (audit-first default). `scope` blocks overlap. `exclusive` blocks the entire repo. Independent of acceptance. |
| `acceptance.allow` | []glob | from item `acceptance_allow` | workspace-sandbox | File globs eligible to auto-apply. Always evaluated, regardless of `lock`. |
| `acceptance.deny` | []glob | from item `acceptance_deny` | workspace-sandbox | File globs that must never auto-apply. Higher precedence than allow. |
| `networkMode` | enum `none\|localhost\|full` | `localhost` | workspace-sandbox | Outbound network policy. `localhost` is the default because scenario CLIs need it. |
| `mode` | enum `tracking\|protected` | `tracking` | workspace-sandbox | Selects the runtime guarantee profile. `protected` adds read-only git and resource limits. |
| `protected.gitReadOnly` | bool | `true` (when `mode=protected`) | workspace-sandbox | Restricts direct git ops to read-only; scenario APIs remain allowed. |
| `protected.execLimits` | object | wired from `ExecutionConfig` | workspace-sandbox | CPU/memory/wall-clock limits enforced in protected mode. |

**State-transition implications:**
- `autoApply=true, manualReview=false`: at run-end, evaluate acceptance against final diff; auto-apply matching changes; transition `Active → Approved` (full) or stay `Active` with partial-approval applied (audit retains the rest).
- `applyOnFailure=true`: same as above on failure terminal; failed run still produces the partial audit trail. Failure reason recorded on the run, not on the sandbox.
- `manualReview=true`: skip auto-apply at run-end; transition to `NeedsReview`; surface to GCT AI Changes tab for operator action.

**Acceptance-rules-only trust model (d3=A):** Auto-apply on failure trusts acceptance rules alone — there is no agent-reported "last coherent checkpoint" or progress-milestone gate. This keeps the agent transparent (F5) and pushes recovery to the downstream `run-level-undo-and-revert` initiative, which provides one-click revert keyed off the run's provenance record. Tightening this contract (e.g., requiring a checkpoint signal) is explicitly out of scope; it would couple the agent's behavior to sandbox mode and violate the transparency principle.

### F10: Required Provenance Categories

Per d3=A, the contract specifies categories (not exact schema) that any provenance record must capture:

| Category | Required fields | Source |
|----------|-----------------|--------|
| Run identity | run id, sandbox id, owner, base commit hash, start/end timestamps | agent-manager |
| Task context | task description / prompt summary, conversation reference, agent profile/model | agent-manager |
| Execution outcome | terminal status (succeeded/failed/cancelled), failure reason, duration | agent-manager |
| Cost | input tokens, output tokens, total cost in USD | agent-manager |
| Change set | file path, change kind (add/modify/delete), accepted/rejected reason | workspace-sandbox |
| Apply event | applied-at timestamp, applied-by run id, target commit hash, partial-apply flag | workspace-sandbox / GCT |

Schema design is left to the implementing items, but every provenance record must populate every category. Missing categories must surface as a record-level warning in GCT AI Changes.

### F11: Source-of-Truth Matrix

| Concern | Authoritative service | Notes |
|---------|----------------------|-------|
| Run identity, lifecycle, cost, conversation | agent-manager | Owns the run record; pushes context into sandbox env vars and provenance metadata |
| Sandbox lifecycle, overlay, locking, acceptance evaluation | workspace-sandbox | Owns state machine and apply mechanics |
| Detection / path resolution from agent's perspective | cli-core (`DetectSandbox`, `ResolveScenarioPath`) | Single shared library; test-genie, scenario runner, scenario CLIs all depend on it |
| Restart/test inside sandbox | scenario runner + test-genie | Must use cli-core; must warn (not silently fall back) on scope mismatch |
| Pending vs committed provenance lifecycle, AI Changes UX | git-control-tower | Owns operator-facing audit surface; consumes provenance from agent-manager and workspace-sandbox |
| Policy / guardrails (long-term) | agent-manager (declared) → workspace-sandbox (enforced) | Today partial in agent-manager only; not the primary contract |

### F12: Validation Matrix (must pass before sandbox-as-default rollout)

| # | Behavior | How to verify | Owning item |
|---|----------|---------------|-------------|
| V1 | `lock=none` allows concurrent sandboxes; acceptance still filters apply | Spawn two overlapping sandboxes; both run; on apply each filters by acceptance independently | `fix/workspace-sandbox-lock-and-acceptance-semantics` |
| V2 | Successful run with `autoApply=true` applies accepted changes without review | Run end transitions sandbox to Approved; GCT shows committed provenance | `execute/agent-manager-sandbox-auto-apply-defaults` |
| V3 | Failed run with `applyOnFailure=true` still applies accepted changes; failure reason recorded | Inject mid-run failure; assert partial apply + failure reason in provenance | `execute/agent-manager-sandbox-auto-apply-defaults` |
| V4 | Coding agent prompt/behavior unchanged across sandboxed vs non-sandboxed runs | Compare agent traces with/without `VROOLI_SANDBOX_*` set; assert no branching on env vars in agent prompt | `execute/sandbox-runtime-e2e-verification` |
| V5 | Scenario restart inside sandbox uses overlay paths; no silent fallback | Run scenario CLI with narrow scope; assert warning emitted, not real-repo fallback | `execute/sandbox-runtime-e2e-verification` |
| V6 | Provenance record populates all six required categories (F10) | Assert GCT pending-by-run record has run identity, task, outcome, cost, change set, and apply event | `execute/gct-pending-ai-provenance-hardening` (downstream initiative) |
| V7 | `networkMode=localhost` allows scenario API calls; `none` blocks them | Run test-genie scenario suite under each mode; assert expected pass/fail | `execute/sandbox-runtime-e2e-verification` |
| V8 | Manual-review opt-in still works | Set `manualReview=true`; assert run lands in `NeedsReview` and operator approve/reject still functions | `execute/agent-manager-sandbox-auto-apply-defaults` |
| V9 | Multi-sandbox concurrent apply ordering is deterministic (no lost writes) | Two sandboxes, overlapping non-acceptance paths, both finish near-simultaneously; assert apply ordering documented and conflict surfaces in GCT | `execute/sandbox-runtime-e2e-verification` |

## Risks & Limitations

| Risk | Impact | Mitigation |
|------|--------|------------|
| `applyOnFailure=true` applies broken partial changes from a failed run | Could land code that doesn't compile or breaks tests | Acceptance rules already filter; per-spawn-surface opt-out; provenance failure reason gives operator one-click revert via downstream `run-level-undo-and-revert` initiative |
| Default flip changes existing operator workflows | Operators relying on review-gated apply see surprise auto-applies | Stage rollout per spawn surface (validation matrix V8 keeps opt-in working); announce default flip with the rollout item |
| Concurrent overlapping sandboxes produce conflicting applies | Lost writes or merge conflicts on apply | `BaseCommitHash` conflict detection already exists; V9 verifies deterministic ordering; rebase support handles long-running cases |
| Provenance enrichment requires changes across 3 services | Cross-service coordination cost | F10 specifies categories not schema, allowing each service to evolve at its own pace; missing-category warnings in GCT keep it visible |
| Overlayfs performance on large repos | Slow run startup or apply | Out of scope here; flag for separate investigation if observed in V4/V5 |
| Backwards compatibility with currently-pending sandboxes when defaults flip | In-flight runs may behave differently after deploy | Defaults apply only to newly-created sandboxes; existing sandboxes keep their original config (workspace-sandbox already persists per-sandbox config) |
| Network-mode enforcement mechanism unclear (F7) | `none` and `protected` mode may not actually constrain network | Treat as a follow-on protected-mode item; do not advertise `none` as a security guarantee until enforcement is verified |

## Actions

### Action 1: Update backlog item — Refine fix/workspace-sandbox-lock-and-acceptance-semantics with contract specifics
- **Kind**: fix
- **Name**: workspace-sandbox-lock-and-acceptance-semantics
- **Changes**:
  - Description: extend to reference F1 + F9, explicitly require that `evaluateAcceptance()` apply rules regardless of `lock` value, and adopt the `lock=none|scope|exclusive` enum from F9.
- **Reason**: F1 + F9 give the fix item an unambiguous target shape. Without this update, the implementer would re-derive the contract.

### Action 2: Update backlog item — Refine execute/agent-manager-sandbox-auto-apply-defaults with config field names
- **Kind**: execute
- **Name**: agent-manager-sandbox-auto-apply-defaults
- **Changes**:
  - Description: adopt `manualReview`, `autoApply`, `applyOnFailure` field names and defaults from F9; explicitly cover the `applyOnFailure=true` failure-path code change; reference V2/V3/V8 from F12 as acceptance criteria.
- **Reason**: d1=A required implementation-ready depth; sibling item must consume the field-level spec.

### Action 3: Update backlog item — Tie execute/sandbox-runtime-e2e-verification to the validation matrix
- **Kind**: execute
- **Name**: sandbox-runtime-e2e-verification
- **Changes**:
  - Description: enumerate V4, V5, V7, V9 from F12 as the verification cases; explicitly include the silent-fallback warning fix from F8 and the multi-sandbox apply-ordering case.
- **Reason**: The verification item exists but currently has no concrete checklist; F12 gives one.

### Action 4: Update backlog item — Sequence execute/agent-manager-default-sandboxing-rollout against validation matrix
- **Kind**: execute
- **Name**: agent-manager-default-sandboxing-rollout
- **Changes**:
  - Description: gate the rollout on V1–V9 passing; sequence spawn surfaces in the order **CLI/test-genie → meta-orchestrator → web console** (per d2=A — lowest blast radius first; web console flips last only after V1–V9 hold across the prior surfaces); each surface gets a per-surface opt-out for `applyOnFailure` for critical paths; reference the operator-comms task for the default flip.
- **Reason**: Today the rollout item has no gating contract or sequence. Tying it to F12 and locking the spawn-surface order make the readiness bar explicit and prevent a too-broad first cutover.

### Action 5: Create backlog item — Provenance enrichment for required categories
- **Kind**: execute
- **Title**: Enrich agent-manager → workspace-sandbox → GCT provenance with required categories
- **Description**: Implement the six required provenance categories from F10 (run identity, task context, execution outcome, cost, change set, apply event). Wire agent-manager to push task summary, conversation ref, model, cost, duration, failure reason into the provenance record consumed by GCT. Surface missing-category warnings in the AI Changes tab. References F3 + F10 + V6.
- **Initiative**: git-control-tower-ai-provenance (upstream initiative — this work belongs there, not here)
- **Priority**: high
- **Effort**: M
- **Depends on**: research/agent-sandbox-auditability-contract
- **Reason for creating (vs updating an existing sibling)**: The audit-foundation initiative has no provenance-enrichment member; the upstream `git-control-tower-ai-provenance` initiative is the correct home (its members own the provenance pipeline). No existing item covers the cross-service category contract, so a new item is needed there. (Note: caller should verify this initiative's membership before creation; if a near-duplicate exists, prefer Update.)

### Action 6: Update document — Capture the contract in scenario docs
- **File**: `scenarios/agent-manager/docs/SANDBOX_CONTRACT.md` (create if absent)
- **Change**: Mirror the Findings + Configuration Surface (F9) + Source-of-Truth Matrix (F11) sections as the canonical operator-facing reference. Link from `scenarios/workspace-sandbox/README.md` and `scenarios/git-control-tower/README.md`.

### Action 7: Update initiative — Note that audit-foundation no longer needs to encode provenance enrichment internally
- **Name**: agent-sandbox-audit-foundation
- **Changes**:
  - Description: clarify that provenance enrichment is owned by upstream `git-control-tower-ai-provenance` per Action 5; this initiative consumes the enriched provenance, it does not produce it.
- **Reason**: Keeps initiative scope tight and prevents duplicate work across the two initiatives.

## Terminology

| Term | Definition |
|------|-----------|
| **Sandbox** | An overlayfs-backed isolated workspace that captures all file changes made during an agent-manager coding run |
| **Scope** | The directory subtree covered by the sandbox overlay, set at creation time |
| **Acceptance rules** | File-level allow/deny patterns that determine which sandbox changes are eligible for auto-apply |
| **Locking** | Mutual exclusion that prevents concurrent sandboxes from targeting the same scope paths |
| **Auto-apply** | Automatically applying accepted sandbox changes to the canonical repo at run completion without manual review |
| **Apply on failure** | Auto-applying accepted sandbox changes even when the run terminates in failure, preserving the audit trail for partial useful work |
| **Manual review** | Opt-in workflow where an operator inspects and approves/rejects sandbox changes before they apply |
| **Tracking mode** | Sandbox mode focused on change isolation and provenance recording; no runtime restrictions beyond overlayfs |
| **Protected mode** | Future sandbox mode that adds network restrictions, read-only git, and resource limits on top of tracking mode |
| **Provenance** | The metadata linking every applied code change back to the agent run, task, conversation, and cost that produced it |

## Limitations

- **High confidence:** F1, F2, F4, F5, F8 are based on direct code reading and are verifiable against the cited file paths and line numbers
- **Medium confidence:** F6, F7 (two-mode model, network enforcement) — config exists but enforcement code may live in deployment/runtime layers not examined
- **Medium confidence:** F9 field names and defaults are derived from the spec's contract decisions; the exact field placement (run config vs sandbox config) may shift slightly during implementation
- **Lower confidence:** F10 provenance categories are derived from the auditability value prop; some categories (e.g., conversation reference) may need a different shape depending on agent-manager's existing data model
- **Not investigated:** Performance impact of overlayfs on large repos, TTL/cleanup edge cases, exact apply-ordering algorithm under simultaneous multi-sandbox completion (called out as V9 to verify), enforcement mechanism for `networkMode=none`
