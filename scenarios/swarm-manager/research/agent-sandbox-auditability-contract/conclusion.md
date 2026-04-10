# Research Conclusion: Canonical Sandbox Auditability Contract

## Research Question

What is the authoritative product and implementation contract for agent-manager sandboxing that makes sandbox the default path for agent-created code changes, with auditability (not security) as the primary value proposition?

**Success criteria:** A written contract with precise defaults, terminology, state transitions, configuration surface, interaction matrix, validation matrix, and rollout recommendation — sufficient for implementation without ambiguity.

## Constraints & Scope

- **In scope:** agent-manager, workspace-sandbox, cli-core, test-genie, scenario runner, Git Control Tower — their interactions during sandboxed coding runs
- **Out of scope:** Non-coding agent runs (chat-only, research), hardware/deployment topology, UI design beyond provenance display
- **Key constraint:** The contract must make sandbox transparent to the coding agent — no prompt or behavior changes based on sandbox mode

## Findings

### F1: Locking and Acceptance Are Currently Conflated (Critical Mismatch)

**Source:** `workspace-sandbox/api/internal/sandbox/service.go`, `evaluateAcceptance()` (lines 1146-1151)

When `NoLock=true` (the current default), the service bypasses acceptance rules entirely — all files are auto-accepted. The code comment reads: "all files are automatically accepted (no acceptance rules apply)."

This directly contradicts the spec requirement that locking and acceptance are independent concerns. The intended contract says:
- **Locking** = mutual exclusion on scope paths (can multiple sandboxes target the same area?)
- **Acceptance** = file-level filtering on approval (which changes are eligible to auto-apply?)

Today they are coupled: `noLock` implies `noAcceptance`. Fixing this is the highest-priority semantic change.

### F2: Auto-Apply Defaults Block the Auditability Path (Critical Mismatch)

**Source:** `agent-manager/api/internal/orchestration/run_executor.go` (lines 1119-1168, 1402-1496)

Current defaults:
- `RequiresApproval = true` (line 1123)
- Default auto-approval strategy: `AutoApproveIfEmpty` — only auto-approves if the sandbox diff is empty

This means every sandboxed run that produces actual code changes lands in `NeedsReview` status. The operator must manually approve. This breaks the desired default: "accepted changes auto-apply at run completion."

The spec wants:
- `RequiresApproval = false` by default (manual review is opt-in)
- Auto-apply accepted changes at run end, including on failure with useful changes
- Acceptance rules (not manual review) gate what gets applied

### F3: Provenance Pipeline Exists But Lacks Execution Context

**Source:** `git-control-tower/api/approved_changes_handler.go`, `AIProvenanceTab.tsx`

The provenance grouping works: changes are grouped by `AgentManagerRunID`, `SandboxID`, and `SandboxOwner`. The `/api/v1/provenance/by-run` endpoint is the source of truth.

**Gaps:**
- No execution metadata visible (conversation context, cost, duration, task description)
- No failure reason tracking — failed runs leave pending changes with no "why"
- No link from provenance to the agent's conversation or task context
- The UI shows file-level changes grouped by run but no drill-through to execution details

For auditability to be the primary value prop, provenance must answer: "What agent, doing what task, at what cost, produced these specific changes and why?"

### F4: Sandbox State Machine Is Well-Designed

**Source:** `workspace-sandbox/api/internal/sandbox/service.go`

The existing state machine is sound:
- `Creating → Active → [Stopped ↔ Active] → Approved/Rejected (terminal)`
- Partial approval keeps sandbox Active (applies subset, preserves rest)
- Conflict detection via `BaseCommitHash` comparison
- Rebase support for long-running sandboxes
- Audit trail records every lifecycle transition
- Pre-teardown hooks for scenario evacuation

This is a solid foundation. The contract should formalize this as-is and layer the new defaults on top.

### F5: Environment Injection and Path Resolution Are Clean

**Source:** `run_executor.go` `SandboxEnvVars()` (lines 909-974), `cli-core/cliutil/sandbox.go` (lines 65-200)

Three env vars (`VROOLI_SANDBOX_ID`, `VROOLI_SANDBOX_MERGED`, `VROOLI_SANDBOX_SCOPE`) are injected at run start. CLI tools detect them via `DetectSandbox()` and transparently redirect path resolution to the overlay. Test-genie and scenario runner both use this.

**This achieves agent transparency** — the coding agent doesn't know it's sandboxed. The contract should formalize these three env vars as the canonical detection interface.

### F6: Two-Mode Model Needs Formal Definition

The spec describes two modes sharing the same auditability contract:
- **Tracking mode:** Overlayfs sandbox for change isolation and provenance. No runtime restrictions beyond the overlay. This is the near-term default.
- **Protected mode:** Adds network restrictions, read-only git, resource limits. Future hardening layer.

Config already has `NetworkMode` and `ExecutionConfig` (resource limits), but these aren't enforced as a coherent "mode." The contract needs to define what each mode guarantees and what it doesn't.

### F7: Network Mode Configuration Exists But Enforcement Is Unclear

**Source:** `workspace-sandbox/api/internal/config/config.go`

The config supports network modes but the enforcement mechanism (iptables? network namespaces? bwrap?) isn't visible in the service layer. The spec requires `none`, `localhost`, and `full` because scenario CLIs depend on localhost APIs.

### F8: Scenario Runner Scope Validation Has a Silent Fallback

**Source:** `scripts/lib/scenario/runner.sh` (lines 253-278)

If sandbox scope is too narrow for lifecycle operations (e.g., `scenarios/my-app/ui` instead of `scenarios/my-app`), the runner silently falls back to the real repo path. This could cause agent changes to be invisible during restart/test without any warning.

## Actions

### Immediate (blocks sandbox-as-default)

1. **Decouple locking from acceptance** — `evaluateAcceptance()` must apply acceptance rules regardless of `NoLock` setting (fix item: `fix/workspace-sandbox-lock-and-acceptance-semantics`)
2. **Flip auto-apply defaults** — Change `RequiresApproval` default to `false`; change default auto-approval strategy from `AutoApproveIfEmpty` to `AutoApprove` for accepted changes (execute item: `agent-manager-sandbox-auto-apply-defaults`)
3. **Define `applyOnFailure` behavior** — When a run fails after producing useful changes, auto-apply accepted changes before transitioning to terminal state

### Near-term (enables auditability value prop)

4. **Enrich provenance metadata** — Store task description, conversation summary, cost, duration, and failure reason alongside run ID in provenance records
5. **Formalize two-mode definitions** — Document tracking mode vs protected mode contracts, what each guarantees
6. **Add scope validation warning** — Scenario runner should warn (not silently fall back) when sandbox scope is too narrow for lifecycle operations

### Future (protected mode hardening)

7. **Network mode enforcement** — Implement `none`/`localhost`/`full` with appropriate isolation mechanism
8. **Read-only git in protected mode** — Restrict direct git operations to read-only; allow controlled scenario APIs
9. **Resource limit enforcement** — Wire ExecutionConfig limits into sandbox runtime

## Terminology

| Term | Definition |
|------|-----------|
| **Sandbox** | An overlayfs-backed isolated workspace that captures all file changes made during an agent-manager coding run |
| **Scope** | The directory subtree covered by the sandbox overlay, set at creation time |
| **Acceptance rules** | File-level allow/deny patterns that determine which sandbox changes are eligible for auto-apply |
| **Locking** | Mutual exclusion that prevents concurrent sandboxes from targeting the same scope paths |
| **Auto-apply** | Automatically applying accepted sandbox changes to the canonical repo at run completion without manual review |
| **Manual review** | Opt-in workflow where an operator inspects and approves/rejects sandbox changes before they apply |
| **Tracking mode** | Sandbox mode focused on change isolation and provenance recording; no runtime restrictions beyond overlayfs |
| **Protected mode** | Future sandbox mode that adds network restrictions, read-only git, and resource limits on top of tracking mode |
| **Provenance** | The metadata linking every applied code change back to the agent run, task, conversation, and cost that produced it |

## Confidence & Limitations

- **High confidence:** Findings F1-F5 are based on direct code reading and are verifiable
- **Medium confidence:** F6-F7 (two-mode model, network enforcement) — config exists but enforcement code may be in layers not examined
- **Low confidence:** Provenance enrichment requirements (F3 actions) — unclear how much metadata workspace-sandbox already stores vs what agent-manager would need to push
- **Not investigated:** Performance impact of overlayfs on large repos, TTL/cleanup edge cases, multi-sandbox concurrent apply ordering
