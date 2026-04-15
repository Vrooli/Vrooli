# Agent Sandbox Audit Foundation

## Mission

Make workspace-sandbox the canonical, default audit path for agent-manager coding runs so Vrooli can correlate repository changes back to the exact agent-manager run, conversation history, cost, and execution context that produced them.

This initiative is intentionally auditability-first, not security-first. Protected execution and stronger runtime restrictions matter, but they are downstream goals. The immediate product value is durable per-run attribution of agent-made changes.

## Product intent captured from the April 9, 2026 workshop conversation

- Sandbox exists primarily to track changes and associate them with specific agent-manager runs.
- The platform wants to encourage sandbox usage everywhere agent-manager is used, because that unlocks Git Control Tower's AI Changes experience and broader auditability.
- The sandbox should feel as seamless as possible. The coding agent should not need prompt changes, special instructions, or altered behavior just because a run is sandboxed.
- The default operator path should not require a separate manual review step.
- Accepted changes should auto-apply by default at the end of a run, even if the run itself fails after producing useful work.
- Acceptance rules should still decide what is eligible to apply.
- Changes outside the accepted area may exist inside the sandbox during execution, but they should not auto-apply by default.
- Locking and acceptance are separate concepts. Most sandboxes should not reserve an exclusive lock on their scope. Multiple sandboxes targeting the same area should be allowed by default, even if overlapping work is discouraged operationally.
- Undo/revert should exist as a later first-class workflow, because auto-apply only feels safe if operators can inspect and back out run-linked work afterward.

## Core contract decisions

### Default lifecycle

- `sandboxed = true` should become the normal execution path once readiness gates are met.
- `manualReview = false` by default.
- `autoApply = true` by default.
- `applyOnFailure = true` by default.
- `acceptanceAllow` and `acceptanceDeny` govern apply eligibility.
- `lock = false` by default and must remain independent from acceptance.

### Tracking vs protected modes

The long-term design supports two sandbox modes:

- `tracking` mode: sandbox is primarily an overlay/change-tracking and apply/review mechanism.
- `protected` mode: same auditability contract, plus real process containment and runtime guardrails.

The rollout order is deliberate:

1. Make tracking mode reliable and seamless.
2. Prove Git Control Tower can surface the resulting provenance clearly.
3. Make sandboxing the default in major spawn surfaces.
4. Harden protected mode over time.

### Network and localhost assumptions

- Network policy eventually needs `none`, `localhost`, and `full`.
- `localhost` is operationally important because scenario CLIs are thin wrappers over local scenario APIs.
- This means sandboxing cannot be designed around "no network at all" as the universal default if the platform expects scenario CLI usage to continue working.

### Git assumptions

- Direct git usage by the agent should eventually be read-only by default in protected mode.
- A reasonable default allowlist is `status`, `diff`, `log`, `show`, and `rev-parse`.
- Commands with side effects such as branching, checkout/reset, commit, merge, rebase, push, pull, clean, and similar operations should not be allowed directly by the agent in protected mode.
- Scenario-mediated git operations remain allowed. The intended model is that trusted scenarios such as Git Control Tower own higher-trust git mutations rather than direct shell git usage from the agent process.

## Current-state findings that motivated this initiative

### What already works or partly works

- Agent-manager already creates workspace-sandbox instances and injects sandbox environment variables.
- Scenario restart flows and cli-core sandbox path resolution already have real sandbox-aware logic.
- Test-genie already contains sandbox-aware path resolution so tests can run against sandbox state instead of only the underlying repo.
- Workspace-sandbox already records applied-change provenance including `agent_manager_run_id`.
- Git Control Tower already has an API/UI path for pending AI provenance grouped by run, including the AI Changes tab.

### Important current gaps

- Agent-manager currently creates the sandboxed merged workspace and then launches Claude/Codex/OpenCode directly in that merged directory. It is not yet using workspace-sandbox's contained execution APIs for the agent process itself.
- Workspace-sandbox currently appears to conflate `noLock` with acceptance bypass. That is incompatible with the intended contract where lock and acceptance are independent.
- Sandboxed runs with `RequiresApproval=false` currently complete without necessarily applying sandbox changes. That breaks the primary auditability goal because the run finishes but the canonical applied-change record is missing.
- Some policy surfaces in agent-manager are partially wired or advisory rather than enforced. Those should not be treated as the primary contract for this initiative.

## Code references captured during investigation

- Sandbox env wiring and lifecycle orchestration:
  - `scenarios/agent-manager/api/internal/orchestration/run_executor.go`
- Runner launch paths:
  - `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go`
  - `scenarios/agent-manager/api/internal/adapters/runner/codex_runner.go`
  - `scenarios/agent-manager/api/internal/adapters/runner/opencode_runner.go`
- Workspace-sandbox semantics and apply behavior:
  - `scenarios/workspace-sandbox/api/internal/sandbox/service.go`
  - `scenarios/workspace-sandbox/api/internal/config/config.go`
- Sandbox-aware scenario path resolution:
  - `packages/cli-core/cliutil/sandbox.go`
  - `scripts/lib/scenario/runner.sh`
  - `scenarios/test-genie/cli/execute/command.go`
  - `cli/commands/scenario/modules/heal.sh`
- Git Control Tower provenance surfaces:
  - `scenarios/git-control-tower/api/approved_changes_handler.go`
  - `scenarios/git-control-tower/api/review_handler_dimensions.go`
  - `scenarios/git-control-tower/api/workspace_sandbox_client.go`
  - `scenarios/git-control-tower/ui/src/components/AIProvenanceTab.tsx`
- Workspace-sandbox provenance storage:
  - `scenarios/workspace-sandbox/api/internal/repository/sandbox_repo.go`
  - `scenarios/workspace-sandbox/api/internal/types/types.go`

## Initiative ordering rationale

### 1. Contract and semantics

The team needs one authoritative contract before more code lands. Otherwise the same ambiguity around approval, locking, acceptance, and apply-on-failure will keep resurfacing.

### 2. Auto-apply and seamlessness verification

The auditability story only works if sandboxed runs reliably produce applied-change provenance, and if restart/test workflows behave as if the sandbox were the active workspace.

### 3. Git Control Tower trust surface

Operators need to see the resulting provenance clearly. Otherwise sandboxing may technically work while still failing the user-facing product goal.

### 4. Default rollout

Only after semantics and visibility are trustworthy should sandboxing become the default in high-traffic spawn surfaces.

### 5. Protected mode later

Containment and enforcement become far more valuable once the auditability/default path is already working, because then protections can mature without threatening adoption of sandboxing itself.

## Non-goals for this initiative

- Do not require agents to change prompts or consciously adapt their behavior based on sandbox mode.
- Do not make locking the default.
- Do not block writes outside acceptance during the run in the default tracking-first model unless a later protected-mode decision explicitly adds that behavior.
- Do not treat advisory runner flags as if they are strong security controls.

## Handoff guidance for workshop and execution agents

- Preserve the auditability-first framing. Do not reorder work into a security-first program unless the user explicitly changes direction.
- Use Git Control Tower correlation and AI Changes clarity as the operator-facing measure of success.
- When evaluating behaviors, test both success and failure paths, because accepted changes should apply even if the run fails.
- Keep the distinction between `manual review`, `auto-apply`, `acceptance`, and `locking` explicit in specs, APIs, and UI copy.
