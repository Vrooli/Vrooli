# Agent Sandbox — Completion + Protected Mode Implementation Plan

> Generated: 2026-04-27. Successor to
> `path:docs/plans/agent-sandbox-audit-cutover-implementation-plan.md` (Phases 3a/3b/4
> complete) and gates the rollout (`agent-sandbox-validation-matrix-readiness.md`).
> This plan finishes everything the two locked initiatives still need —
> `agent-sandbox-audit-foundation` (3 of 10 items pending) **and**
> `protected-agent-sandboxing` (3 of 3 items pending) — plus all greenfield
> cleanup the user requires before either can be considered "truly complete".

---

## 1. Purpose

Drive `agent-sandbox-audit-foundation` and `protected-agent-sandboxing` from
"behavior cutover landed" to "fully shipped, default on, legacy purged,
protected-mode real" — without leaving compatibility shims, vestigial fields,
dead policy seams, or `// removed in Phase 3b` comment trails behind. The user
considers an initiative complete only when:

1. The required outcomes in each backlog item are met.
2. There is **no legacy / dead / wrapper code** left in the touched scenarios.
3. `go vet`, `golangci-lint run`, `tsc --noEmit`, and `go test ./...` are all
   green on agent-manager and workspace-sandbox (preexisting issues count).
4. Live wire smokes against restarted services round-trip the new fields.
5. Default-flip is observable in the UI and swarm-manager queue spawn paths.

---

## 2. Required Reading

Run these before touching any code; they encode the existing canon:

```bash
prompt-manager skill read implementation-plan-authoring \
  cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Locked contract (source of truth — **do not redesign**):

```bash
cat /home/matthalloran8/Vrooli/scenarios/swarm-manager/research/agent-sandbox-auditability-contract/conclusion.md
cat /home/matthalloran8/Vrooli/scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md
cat /home/matthalloran8/Vrooli/scenarios/workspace-sandbox/docs/EXECUTION_MODES.md
```

Direct predecessors (resume context, not authority):

```bash
cat /home/matthalloran8/Vrooli/docs/plans/agent-sandbox-audit-cutover-implementation-plan.md
cat /home/matthalloran8/Vrooli/docs/plans/agent-sandbox-validation-matrix-readiness.md
```

Backlog items being closed by this plan:

```bash
swarm-manager backlog get --kind execute --name agent-manager-default-sandboxing-rollout
swarm-manager backlog get --kind execute --name sandbox-provenance-schema-version-shared-package
swarm-manager backlog get --kind execute --name agent-manager-spawn-surface-conversation-id-population
swarm-manager backlog get --kind execute --name protected-sandbox-agent-launch
swarm-manager backlog get --kind execute --name protected-sandbox-git-and-network-guardrails
swarm-manager backlog get --kind fix   --name protected-sandbox-policy-enforcement-surface
```

---

## 3. Greenfield Constraint (HARD RULE)

> **Repeated in §13 Definition of Done. Non-negotiable.**

- No backwards-compatibility shims, no dual-name fields, no `// removed in
  Phase X` comment trails, no `_unused` aliases, no DB read-time normalization
  layers preserved for "legacy persisted runs". Old fields are **deleted**, the
  DB column is **dropped**, and old comments are **deleted**, not annotated.
- No code paths that exist only to keep an unused interface alive (e.g.
  `policy.Decision.RequiresApproval`, `policy.CanAutoApprove`,
  `AcceptanceConfig.AutoApprove`/`AutoReject`).
- No "deprecated but kept" exports in scenarios that consume agent-manager.
- Reserved proto fields stay reserved (they're a wire-compat necessity, not a
  shim) — but every Go struct field, JSON key, DB column, TypeScript prop, and
  comment that referenced the removed surface is purged.
- The contract's *forward-looking* fields (`runOutcome`, `state`,
  `conversationId`, `costUsd`, `manualReview`) must be **persisted end-to-end**,
  not stuck at the wire boundary.

If you find yourself writing a "for migration purposes" branch, stop and ask.

---

## 4. Problem Statement

### 4.1 Audit-foundation — what's left

The cutover plan (predecessor) landed Phases 3a/3b/4 + E2E (auto-apply at run
end, manual-review TTL GC, 9-behavior validation matrix). The gate-flip and
the data-layer follow-throughs were intentionally deferred:

| # | Pending item | Concrete gap |
|---|---|---|
| 1 | `agent-manager-default-sandboxing-rollout` | UI `QuickRunDialog` defaults `runMode = RunMode.IN_PLACE` (`path:scenarios/agent-manager/ui/src/components/QuickRunDialog.tsx:167,227`); `useTasksRunDialogState` defaults `SANDBOXED`. No adoption metric. No "non-sandboxed escape hatch is intentional, not accidental" copy. swarm-manager queue spawn (`path:scenarios/swarm-manager/api/internal/agentmanager/service.go:271-281`) never sets `RunMode` — relies on agent-manager's `valueOrDefault(req.RunMode, RunModeSandboxed)` default at `service.go:1044,1063`. That default is correct, but it's silent — there's no "intentional in-place" distinction. |
| 2 | `sandbox-provenance-schema-version-shared-package` | Contract fields `runOutcome`, `state`, `conversationId`, `costUsd` round-trip on the wire (Phase 3b), but workspace-sandbox does NOT persist them. Provenance readers see empty values. Decision 3 in the contract names a shared schema-version package; it doesn't exist yet. |
| 3 | `agent-manager-spawn-surface-conversation-id-population` | `Run.ConversationID` / `Run.ParentRunID` are populated only by the inheritance fallback. swarm-manager queue, web-console, cron, direct-CLI never set them explicitly. Item description also calls for "DB read-time normalization for legacy persisted runs that still carry `Acceptance.AutoApprove==true` / `RequiresApproval`" — under the greenfield constraint, that requirement is **inverted**: drop the column and any persisted legacy values via migration; do not normalize. |

### 4.2 Audit-foundation — legacy code still in the tree

Code scan (2026-04-27) shows:

| Site | Status | Disposition |
|---|---|---|
| `path:scenarios/agent-manager/api/internal/policy/interface.go:89-90` `Decision.RequiresApproval` | Returned by Evaluator but never consumed (not on `Run`, not on `RunConfig`, not on `Decision` callers' apply paths) | **Delete the field**. Update `policy.Decision`, `policy/interface_test.go`, and any constructor that sets it. |
| `path:scenarios/agent-manager/api/internal/database/repository.go:94,109-115,170-173,214,227,230,301` | DB still has `requires_approval` column; read-time shim sets `manualReview=true` if true; writes always set `false` | **Drop the column** via migration; remove the shim and all references. |
| `path:scenarios/agent-manager/api/internal/database/schema.sql:24` `requires_approval INTEGER DEFAULT 1` | Schema source-of-truth still defines it | Remove from `schema.sql`; the migration in `connection.go` does the production drop. |
| `path:scenarios/agent-manager/api/internal/database/connection.go` | No `ensureProfileRequiresApprovalDrop` migration exists yet | Add one (idempotent `ALTER TABLE … DROP COLUMN IF EXISTS`). |
| `path:scenarios/agent-manager/api/internal/domain/types.go:53,690,1379` | `// RequiresApproval was removed in agent-sandbox-audit-foundation Phase 3b.` comments + `InPlaceRequiresApproval *bool` field on `RunConfigOverrides` (line 1379) | Comments: delete. The `InPlaceRequiresApproval` override field: investigate — if not consumed, delete; if consumed, rename to `InPlaceManualReview`. |
| `path:scenarios/agent-manager/api/internal/domain/types_test.go:220` | Test comment references removed field | Delete the comment; verify the test still asserts the right behavior. |
| `path:scenarios/agent-manager/api/internal/orchestration/{service.go:230, investigation.go:1349, run_executor_test.go:1833, sandbox_config_test.go:3-4}` | Comment trails | Delete. |
| `path:scenarios/agent-manager/api/internal/handlers/handlers.go:1366` | Comment trail | Delete. |
| `path:scenarios/agent-manager/ui/src/{types.ts:112,146; hooks/useApi.ts:314,389; pages/ProfilesPage.tsx:656; pages/TasksPage.tsx:1191}` | Comment trails | Delete. |
| `path:scenarios/ecosystem-manager/api/pkg/agentmanager/service.go:372` and `path:scenarios/app-issue-tracker/api/internal/agentmanager/service.go:216` | Comment trails | Delete. |
| `path:scenarios/agent-manager/api/internal/orchestration/sandbox_config_test.go` (entire file header comment) | Refers to "retired" fields | Either delete the file (if tests duplicate `auto_approval_test.go`) or replace the header with the live test purpose. |
| `path:scenarios/swarm-manager/api/internal/settings/adapters.go:17` `LoadAgentSettings(...) (... requiresApproval bool ...)` and `path:scenarios/swarm-manager/cli/cmd_settings.go:39,72`, `path:scenarios/swarm-manager/ui/src/{services/settings-service.ts:45,80,129; services/proto/settings-contracts.ts:58; types/settings.ts:52; components/settings/ExecutionTab.tsx:177,209,214}` | swarm-manager exposes a global `agentRequiresApproval` setting that no longer maps to anything in agent-manager (the run-level field was deleted) | **Delete the setting end-to-end** (CLI, API, UI, proto, store) OR — if user wants a global "always set ManualReview=true on spawned runs" toggle — rename to `agentManualReview` and wire it through `AgentService.{SpawnResearch,SpawnBacklog,SpawnInitiative}` to populate `SandboxConfig.ManualReview` on the `CreateRunRequest`. **Default to delete** under the wrap-not-use principle: per-profile `SandboxConfig.ManualReview` is the right grain. |

### 4.3 Workspace-sandbox — legacy code still in the tree

| Site | Status | Disposition |
|---|---|---|
| `path:scenarios/workspace-sandbox/api/internal/types/types.go:128-136` `AcceptanceConfig.AutoApprove`, `AutoReject` | Set in JSON, never read by `apply-at-run-end` or any other handler | **Delete both fields**. Verify no callers (search shows zero non-test callers). |
| `path:scenarios/workspace-sandbox/api/internal/policy/{policy.go,approval.go,policy_test.go}` `ApprovalPolicy` interface + `DefaultApprovalPolicy` + `RequireHumanApprovalPolicy` + `CanAutoApprove` | Interface and impls only have callers inside the policy package (tests) | **Delete the interface and both impls** along with their tests. The new contract makes the agent-manager-driven `apply-at-run-end` the single decision point; sandbox-side approval policy is dead. |
| `path:scenarios/workspace-sandbox/api/internal/config/config.go:188-198,435-436,523-524` `AutoApproveThresholdFiles/Lines` + env var wiring | No production caller after the policy interface is removed | **Delete the config fields, defaults, and `WORKSPACE_SANDBOX_AUTO_APPROVE_*` env vars.** |
| `path:scenarios/workspace-sandbox/api/internal/logging/logging_test.go:386,400` `PolicyValidation("auto_approve", …)` test labels | Tests for `Logger.PolicyValidation` use `"auto_approve"` as a string label | If the policy code is removed and `PolicyValidation` has no remaining production callers using that label, switch the label to a still-used policy event (e.g. `"acceptance_deny"` or `"manual_review_ttl"`) so the test mirrors a real call site. |
| `path:scenarios/workspace-sandbox/api/internal/sandbox/manual_review_ttl_test.go` (5 sites) | gofumpt formatting violations introduced by Phase 4 | `gofumpt -w` |
| `path:scenarios/workspace-sandbox/api/internal/types/{types.go:279,281, auditability_contract_test.go:43}` | gofumpt formatting | `gofumpt -w` |

### 4.4 Lint / format issues

`golangci-lint run ./...` (2026-04-27, both scenarios), only gofumpt issues:

```
agent-manager:
  internal/database/repository.go:169, 175
  internal/orchestration/service.go:234

workspace-sandbox:
  internal/sandbox/manual_review_ttl_test.go:31, 34
  internal/types/types.go:279, 281
  internal/types/auditability_contract_test.go:43
```

All trivially fixable with `gofumpt -w` on the listed files. Plan addresses
them as a final-phase step alongside the comment-cleanup pass.

### 4.5 Protected-agent-sandboxing — gap analysis

Backlog item `execute/protected-sandbox-agent-launch` requires the coding agent
process itself to launch through `workspace-sandbox` execution APIs (`/exec`,
`/processes`, `/interactive`) instead of being spawned directly on the host
inside the merged overlay. Current state:

| Surface | Current implementation | Gap |
|---|---|---|
| `path:scenarios/agent-manager/api/internal/adapters/runner/claude_code.go` | Spawns `claude` directly on the host with `cmd.Dir = sandbox.MergedPath` and host env injection | Must dispatch through `workspace-sandbox` `/exec` (sync) or `/processes` (async with streaming) when `SandboxConfig.Mode == SandboxModeProtected`. |
| `path:scenarios/agent-manager/api/internal/adapters/runner/codex_runner.go` | Same shape; also branches on `cfg.NetworkAccess.Effective()` to compute Codex `--full-auto` flags | Network-mode flag handling stays, but launch path forks on `Mode`. Codex `--full-auto` is the *current* enforcement mechanism for `none`; it remains for tracking mode (per memory). |
| `path:scenarios/agent-manager/api/internal/adapters/runner/opencode_runner.go` | Direct host launch | Same. |
| `path:scenarios/workspace-sandbox/api/internal/handlers/{process.go,interactive.go}` | Already implements `/processes` async and `/interactive` PTY-over-WebSocket | No change required; this is the seam. |
| `path:scenarios/workspace-sandbox/api/internal/driver/bwrap.go:480-497` | Switches `NetworkAccess` "none"/"localhost"/"full" via bwrap network namespace flags | Already supports the three modes. |
| `path:scenarios/agent-manager/api/internal/adapters/sandbox/interface.go` | Provider has `ApplyAtRunEnd`, `Create`, etc., but no `ExecProcess` / `LaunchInteractive` | Add provider methods that proxy to workspace-sandbox `/processes` and `/interactive`. |

For `execute/protected-sandbox-git-and-network-guardrails`:

| Surface | Current implementation | Gap |
|---|---|---|
| Git allowlist for direct agent calls | None — agent runs git directly with whatever the merged overlay contains | Needs a `git`-shim/intercept layer or, more cleanly, a workspace-sandbox-side allow/deny enforcement on `/exec` argv when `SandboxBehavior.Protected.GitAllowlist` is set. Locked allowlist (per backlog item): `status, diff, log, show, rev-parse`. Block: `branch, commit, checkout, reset, rebase, merge, push, pull, clean`. |
| Network mode enforcement | Codex runner sets `--full-auto` to bake-in network restrictions; bwrap sets network namespace per `NetworkAccess`. **No path actually wires `NetworkMode` from `SandboxConfig` into the runner today** — runners read `cfg.NetworkAccess`, which is the agent-profile-level field, not the per-run sandbox-config one. | Wire `SandboxConfig.NetworkMode` (added in Phase 3b cutover) through to the runner cfg in protected mode; in tracking mode preserve current Codex-flag behavior (per memory note). |
| Denial UX | None | Workspace-sandbox `/exec` should return a structured 403 with `reason="git-verb-blocked"` and a human message; agent-manager surfaces it as a tool-call error event. |

For `fix/protected-sandbox-policy-enforcement-surface`:

| Surface | Current implementation | Gap |
|---|---|---|
| `path:scenarios/agent-manager/api/internal/policy/**` `Decision.RequiresApproval`, `Decision.RequiresSandbox`, `Decision.AllowedPaths` (et al.) | Decision struct returns `RequiresApproval` (now dead), `RequiresSandbox`, `EffectiveTimeout`, `EffectiveMaxFiles`, `EffectiveMaxSize` | Decision: which controls become **real protected-mode enforcement** vs. **deleted as misleading**. Locked stance from the backlog item: enforce path/resource boundaries at the sandbox layer; remove anything the runtime cannot enforce. |
| `policy.Evaluator` interface | Stub-quality — `RequiresApproval` is dead-on-arrival now | Slim the interface to only what runners + sandbox enforce; document the seam. |
| `Profile.AllowedPaths` / `DeniedPaths` | Stored in DB and `Profile`, passed to runners as advisory env (not enforced) | In protected mode, push these to workspace-sandbox `Behavior.AcceptanceAllow/Deny` (already enforced in apply-at-run-end) and bwrap mount filters where applicable. |

### 4.6 Coupling rationale

These two initiatives must land in this plan together because:

- The audit-foundation cutover left `SandboxConfig.Mode` as a typed enum
  (`Tracking | Protected`) but only `Tracking` is wired. Until the protected
  path is real, the field is theoretically out-of-state.
- The default-rollout flip surfaces sandboxed runs everywhere — making the
  protected-mode launch-path unpaved more visible (any user who chooses
  protected gets silent-fall-through to tracking).
- The legacy `policy.Decision.RequiresApproval` deletion in Phase A overlaps
  with the policy-surface cleanup in `fix/protected-sandbox-policy-enforcement-surface`.
  Doing them in different PRs would mean churning the policy package twice.

---

## 5. Scope

### In scope

- All 6 pending backlog items across both initiatives.
- Greenfield removal of the legacy code surfaces enumerated in §§4.2-4.3.
- Lint/format cleanup in §4.4.
- Persistence migration for `runOutcome` / `state` / `conversationId` / `costUsd`
  on workspace-sandbox provenance and the `requires_approval` column drop on
  agent-manager.
- A shared schema-version package (`path:packages/sandbox-provenance`) consumed by
  both workspace-sandbox and the GCT pending-AI hardening initiative.
- Adoption metrics on the agent-manager `/metrics` endpoint and a swarm-manager
  `swarm-manager stats sandbox-adoption` CLI subcommand.
- End-to-end live wire smokes against restarted services.

### Out of scope

- `gct-pending-ai-provenance-hardening` (sibling initiative on a different
  branch) — Phase E coordinates the schema-version package contract with it
  but does not implement GCT changes.
- Cron-driven and direct-CLI spawn-surface flips — explicitly deferred per the
  default-rollout backlog item description (Phase D handles UI + swarm-manager
  queue only).
- Tool-level `inbox.ToolMeta.RequiresApproval` (`path:packages/proto/schemas/agent-inbox/v1/domain/tool.proto:152`)
  — different concept, governs per-tool human-in-the-loop, **not** a legacy of
  this initiative. Untouched.

---

## 6. Current Technical Context

Key files and what they own (use these as your map; `Read` them before
editing):

```
scenarios/agent-manager/api/internal/
  domain/types.go                  # SandboxMode, SandboxConfig, RunConfig, RunMode
  domain/decisions.go              # ResolvedConfig, decision wiring
  domain/validation.go             # validateSandboxMode + SandboxConfig validation
  policy/interface.go              # Evaluator, Decision (RequiresApproval lives here, dead)
  database/schema.sql              # requires_approval INTEGER DEFAULT 1 (to drop)
  database/connection.go           # migrations (ensureProfile* helpers)
  database/repository.go           # profileRow + read-time shim (to delete)
  orchestration/service.go         # CreateRunRequest, valueOrDefault, RunMode default
  orchestration/run_executor.go    # applyAtRunEnd (Phase 3b shipped)
  orchestration/investigation.go   # apply-investigation profile (manual_review)
  adapters/sandbox/interface.go    # Provider seam (add Exec/Interactive)
  adapters/sandbox/workspace_sandbox.go # WorkspaceSandboxProvider
  adapters/runner/{claude_code,codex_runner,opencode_runner}.go # protected-mode launch path
  protoconv/{convert,entities}.go  # SandboxConfig <-> proto

scenarios/workspace-sandbox/api/internal/
  types/types.go                   # SandboxBehavior, AcceptanceConfig (AutoApprove/AutoReject to delete)
  sandbox/{service,lifecycle}.go   # apply-at-run-end + manual-review TTL
  policy/{policy,approval}.go      # ApprovalPolicy interface (to delete)
  config/config.go                 # AutoApproveThreshold* + WORKSPACE_SANDBOX_AUTO_APPROVE_* env (to delete)
  handlers/{process,interactive,sandbox}.go # /exec /processes /interactive (protected-mode seam)
  driver/bwrap.go                  # NetworkAccess + mount/syscall enforcement
  database/...                     # provenance schema (add runOutcome/state/conversationId/costUsd columns)

scenarios/swarm-manager/api/internal/agentmanager/service.go
  # SpawnResearch / SpawnBacklog / SpawnInitiative — populate ConversationID + ParentRunID

scenarios/swarm-manager/{api,cli,ui}/...settings...
  # Delete agentRequiresApproval end-to-end

scenarios/agent-manager/ui/src/
  components/QuickRunDialog.tsx    # default RunMode flip (167, 227)
  hooks/useTasksRunDialogState.ts  # already SANDBOXED — verify
  pages/{TasksPage,ProfilesPage}.tsx
  components/RunDetail.tsx         # surface SandboxMode + adoption visual

packages/proto/schemas/agent-manager/v1/domain/
  profile.proto                    # reserved markers (keep)
  types.proto                      # SandboxConfig fields (mode/manual_review/auto_apply/apply_on_failure/network_mode/no_lock)

packages/sandbox-provenance         # NEW — shared schema-version package
```

Test infrastructure:

- agent-manager: `go test ./...` runs ~1 minute. No external dependencies.
- workspace-sandbox: `go test ./...` runs ~3 seconds. No external dependencies.
- swarm-manager: `go test ./...` runs ~6 minutes.
- agent-manager UI: `npx tsc --noEmit` runs in <60s.
- Lint: `golangci-lint run ./...` per scenario (~30s each).
- Live smoke: `vrooli scenario restart agent-manager workspace-sandbox` then
  `curl` against `/health` and `/api/v1/sandboxes/{id}/apply-at-run-end`.

---

## 7. Target End State

When this plan is complete:

1. `swarm-manager initiatives get --name agent-sandbox-audit-foundation` reports
   10/10 completed; `--name protected-agent-sandboxing` reports 3/3 completed.
2. `git grep -n 'RequiresApproval\|AutoApprove\|AutoReject\|DisableAutoApproveIfEmpty'`
   inside `path:scenarios/agent-manager`, `path:scenarios/workspace-sandbox`,
   `path:scenarios/swarm-manager`, `path:scenarios/ecosystem-manager`,
   `path:scenarios/app-issue-tracker`, `path:scenarios/scenario-auditor`,
   `path:scenarios/test-genie` returns **only**:
   - per-tool `inbox.ToolMeta.RequiresApproval` references in `toolregistry/`
     directories (legitimate, different concern), AND
   - `reserved` markers in `path:packages/proto/schemas/agent-manager/`.
3. `path:scenarios/agent-manager/api/internal/database/schema.sql` no longer has the
   `requires_approval` column; production DBs have it dropped via migration
   added to `connection.go`.
4. `golangci-lint run ./...` is clean on agent-manager and workspace-sandbox.
5. `go test ./...` is green on agent-manager, workspace-sandbox, swarm-manager,
   ecosystem-manager, app-issue-tracker, scenario-auditor, test-genie.
6. Agent-manager UI typecheck (`npx tsc --noEmit`) green; the `QuickRunDialog`
   defaults `runMode` to `RunMode.SANDBOXED`; in-place is reachable but
   labelled as the "operator escape hatch".
7. swarm-manager queue spawn (`AgentService.{SpawnResearch,SpawnBacklog,SpawnInitiative}`)
   explicitly populates `ConversationID` and `ParentRunID` on every
   `CreateRunRequest`; agent-manager no longer has to fall back to inheritance.
8. workspace-sandbox persists `runOutcome`, `state`, `conversationId`,
   `costUsd` on provenance records via the `path:packages/sandbox-provenance`
   schema-version package; readers (web-console + GCT) see populated fields.
9. Protected mode is real:
   - `SandboxConfig.Mode == SandboxModeProtected` causes runners to launch
     through workspace-sandbox `/processes` (not host exec).
   - Direct `git` calls inside protected sandboxes are limited to the
     allowlist (`status, diff, log, show, rev-parse`); other verbs return a
     structured denial.
   - `NetworkMode` from `SandboxConfig` flows to runner + bwrap; the three
     modes (`none`, `localhost`, `full`) are observably different on a smoke.
   - `policy.Decision.RequiresApproval`, `policy.CanAutoApprove`, and
     `AutoApproveThreshold*` are removed from the codebase.
10. Adoption metrics emit on agent-manager `/metrics`:
    `agent_manager_runs_total{run_mode,sandbox_mode,manual_review}` and
    `agent_manager_runs_with_provenance_total`. swarm-manager `stats` exposes
    them via `swarm-manager stats sandbox-adoption`.
11. `path:docs/plans/agent-sandbox-validation-matrix-readiness.md` is updated to
    mark all 9 contract behaviors ✅ (the 3 🟡 integration tests land in
    Phase D); rollout flip is recorded as completed.

---

## 8. Implementation Strategy (phased)

> **Order is load-bearing.** Each phase is a coherent landable unit.
> Run `go test`, `golangci-lint`, and `tsc --noEmit` between phases.
> Restart `agent-manager` and `workspace-sandbox` whenever wire-shape
> structs change, then live-smoke `/health` + `apply-at-run-end`.

### Phase A — Greenfield purge (audit-foundation cleanup)

Goal: delete all dead/legacy/comment-trail surfaces enumerated in §§4.2-4.3
in one atomic landing so the codebase is clean before the rollout flip.

A1. **Delete `policy.Decision.RequiresApproval` and all callers.**
- `path:scenarios/agent-manager/api/internal/policy/interface.go:89-90` — remove
  the field.
- `path:scenarios/agent-manager/api/internal/policy/interface_test.go:28,41` —
  remove the assertion.
- `git grep` shows no other consumers (verify before merging).

A2. **Drop the `requires_approval` DB column.**
- Add `ensureProfileRequiresApprovalDropped(ctx)` to
  `path:scenarios/agent-manager/api/internal/database/connection.go` (idempotent
  `ALTER TABLE agent_profiles DROP COLUMN IF EXISTS requires_approval`).
- Wire it into the migration sequence after
  `ensureProfileNetworkAccessColumn`.
- Remove `RequiresApproval` from the `profileRow` struct, the read shim
  (lines 109-115), the write defaults (lines 170-173), and all SQL strings
  (lines 214, 227, 230, 301).
- Remove the column from `path:scenarios/agent-manager/api/internal/database/schema.sql:24`.

A3. **Delete the `// removed in Phase 3b` comment trails.**
- Files in §4.2 row 4 and rows 5-9 + UI rows. Comment-only deletes; the
  removed-field statement is implicit once the field is gone.

A4. **Delete `RunConfigOverrides.InPlaceRequiresApproval`** at
`path:scenarios/agent-manager/api/internal/domain/types.go:1379` if `git grep`
shows zero consumers. (If there are consumers, rename to `InPlaceManualReview`
and wire it through to `SandboxConfig.ManualReview`. Strong prior is
**delete** — the override predates the contract.)

A5. **Delete swarm-manager `agentRequiresApproval`** end-to-end:
- `path:scenarios/swarm-manager/cli/cmd_settings.go:39,72`
- `path:scenarios/swarm-manager/api/internal/settings/adapters.go:17` (slim the
  function signature)
- `path:scenarios/swarm-manager/api/internal/settings/adapters_test.go:54`
- `path:scenarios/swarm-manager/ui/src/services/settings-service.ts:45,80,129`
- `path:scenarios/swarm-manager/ui/src/services/proto/settings-contracts.ts:58`
- `path:scenarios/swarm-manager/ui/src/types/settings.ts:52`
- `path:scenarios/swarm-manager/ui/src/components/settings/ExecutionTab.tsx:177,209,214`
- Settings proto schema if the field is defined there — mark `reserved`.
- Update any swarm-manager docs/skills that reference `agent_requires_approval`.

A6. **Delete `workspace-sandbox` `AcceptanceConfig.AutoApprove`/`AutoReject`** at
`path:scenarios/workspace-sandbox/api/internal/types/types.go:128-136`.

A7. **Delete `workspace-sandbox.policy.ApprovalPolicy` interface and impls.**
- `path:scenarios/workspace-sandbox/api/internal/policy/policy.go` — remove the
  `CanAutoApprove` line from the interface; if the interface becomes empty,
  delete the file.
- `path:scenarios/workspace-sandbox/api/internal/policy/approval.go` — delete.
- `path:scenarios/workspace-sandbox/api/internal/policy/policy_test.go` — delete
  the `TestDefaultApprovalPolicy_CanAutoApprove`,
  `TestRequireHumanApprovalPolicy_CanAutoApprove` cases (and the helper at
  line 918) but keep tests for any policy responsibilities that survive
  (acceptance-allow/deny enforcement).
- `path:scenarios/workspace-sandbox/api/internal/config/config.go:188-198,435-436,523-524`
  — remove the `AutoApproveThreshold*` fields, defaults, and env-var loaders.
- `path:scenarios/workspace-sandbox/api/internal/config/config_test.go:72-76` —
  remove asserts.
- `path:scenarios/workspace-sandbox/api/internal/logging/logging_test.go:386,400`
  — change the `"auto_approve"` test label to a still-live event label
  (e.g. `"acceptance_deny"` or `"manual_review_ttl"`).

A8. **`gofumpt -w`** on every file touched in A1-A7 plus the §4.4 list.

A9. **Verify** — run `go vet`, `golangci-lint run`, and `go test ./...` on
agent-manager, workspace-sandbox, swarm-manager, ecosystem-manager,
app-issue-tracker, scenario-auditor, test-genie. UI typecheck. All green.

A10. **Restart** both scenarios and live-smoke
`POST /api/v1/sandboxes/{id}/apply-at-run-end` to confirm the wire shape
still round-trips.

### Phase B — Schema-version package + provenance persistence (Phase 5 / shared package)

Closes `execute/sandbox-provenance-schema-version-shared-package`.

B1. **Create `path:packages/sandbox-provenance`** as a Go module + TS package:
- `path:packages/sandbox-provenance/go/schema.go` exporting `SchemaVersion = "1.0.0"`,
  the four canonical field names (`runOutcome`, `state`, `conversationId`,
  `costUsd`), and a `Validate(record map[string]any) error` helper.
- `path:packages/sandbox-provenance/ts/schema.ts` mirroring the constants for the
  web-console and GCT UI consumers.
- Tag the package version 1.0.0 in `path:packages/sandbox-provenance/go/go.mod`
  and root `package.json` workspaces.

B2. **Coordinate with `gct-pending-ai-provenance-hardening`** by writing a
short coordination note at
`path:packages/sandbox-provenance/COORDINATION.md` describing the contract
(field names, types, schema version) so the GCT initiative can vendor the
package on its own branch.

B3. **Wire workspace-sandbox to persist** `runOutcome`, `state`,
`conversationId`, `costUsd`:
- Add columns to the provenance table (likely
  `path:scenarios/workspace-sandbox/api/internal/database/schema.sql`); add an
  idempotent migration in `connection.go` style.
- Update `apply-at-run-end` handler to write all four fields when present
  on the request body.
- Update the provenance read path to surface them (both REST and any
  internal pollers).
- Validate against `sandbox-provenance/go/Validate` on write.

B4. **Tests**:
- Unit: `path:packages/sandbox-provenance/go/schema_test.go` — round-trip and
  `Validate` rejection cases.
- Integration: `path:scenarios/workspace-sandbox/api/internal/sandbox/apply_at_run_end_persistence_test.go`
  — POST → DB row contains all four populated fields; subsequent GET
  surfaces them.

### Phase C — Spawn-surface ConversationID/ParentRunID population

Closes `execute/agent-manager-spawn-surface-conversation-id-population`
(modulo the inverted "DB normalization" requirement, which Phase A already
addressed by deleting the column).

C1. **swarm-manager queue spawn** —
`path:scenarios/swarm-manager/api/internal/agentmanager/service.go`:
- All three `runReq := &apipb.CreateRunRequest{...}` sites (lines 271, 332,
  415) must populate `ConversationId` and `ParentRunId`.
- ConversationID source: derive from the queue item's
  `correlation_id` if set, else mint `uuid.New()`. ParentRunID:
  for re-runs, the originating run's ID; otherwise nil.

C2. **agent-manager web-console spawn** — `path:scenarios/agent-manager/ui/src/`
hooks that POST `CreateRunRequest`:
- `useApi.ts` and `useTasksRunDialogState.ts`: ensure
  `conversationId` and `parentRunId` are populated from a `useConversationId()`
  hook that pulls from session storage (mint on first new-conversation).
- `QuickRunDialog.tsx`: thread these through `request` payload.

C3. **Cron-driven spawn** — search for `cron`-prefixed Go files in
`path:scenarios/agent-manager/api/internal` (none exist as of inventory; cron is
part of the swarm-manager backlog meta-orchestrator). If swarm-manager has a
cron path that spawns runs, populate fields there too.

C4. **Direct CLI** — `path:scenarios/agent-manager/cli/`: any `run create`
subcommand must accept `--conversation-id` and `--parent-run-id` and default
to a freshly-minted conversation ID.

C5. **agent-manager** — once all spawn surfaces populate explicitly, remove
the inheritance fallback in
`path:scenarios/agent-manager/api/internal/orchestration/service.go` (search for
`ConversationID`/`ParentRunID` defaulting). The fallback was scaffolding;
its removal is greenfield-aligned.

C6. **Tests**:
- swarm-manager `service_test.go`: assert `ConversationId` and `ParentRunId`
  are non-empty on every spawn helper.
- agent-manager: an integration test creating a run via the CLI surface and
  asserting the persisted run row carries both fields.

### Phase D — Default rollout flip + adoption metrics + readiness checklist closeout

Closes `execute/agent-manager-default-sandboxing-rollout`.

D1. **UI default flip** —
`path:scenarios/agent-manager/ui/src/components/QuickRunDialog.tsx:167,227`:
flip the default in both `useState` initializers from `RunMode.IN_PLACE`
to `RunMode.SANDBOXED`. `useTasksRunDialogState.ts:33` is already correct.

D2. **UI copy** — update the "Run Mode" Select label and option subtitles
to reframe sandboxed as the normal audit path (use the contract's
`Behavior.Mode == tracking` framing) and in-place as "operator escape hatch
— bypasses provenance and review queue".

D3. **swarm-manager queue spawn** — `service.go` (Phase C also touches this
file): explicitly set `RunMode: RunMode_RUN_MODE_SANDBOXED.Enum()` even
though it's the default; this makes the intent legible in logs.

D4. **Compatibility safeguards** — for any spawn surface that cannot use
sandboxed mode (e.g. scenarios on filesystems that bwrap can't overlay-mount,
which exists for some FUSE setups — check `driver/fuse_overlayfs.go`), emit a
structured `WARN` event ("falling back to in_place: <reason>") rather than
silently downgrading. Drop the silent fallback if any.

D5. **Adoption metrics** —
`path:scenarios/agent-manager/api/internal/orchestration/run_executor.go` (or a
new `metrics/sandbox_adoption.go`): expose Prometheus counters
- `agent_manager_runs_total{run_mode,sandbox_mode,manual_review}`
- `agent_manager_runs_with_provenance_total`
on the existing `/metrics` endpoint.

D6. **swarm-manager stats** — add
`swarm-manager stats sandbox-adoption` subcommand
(`path:scenarios/swarm-manager/cli/cmd_stats.go`) that scrapes the counters and
prints a human-readable adoption breakdown.

D7. **Readiness checklist closeout** — update
`path:docs/plans/agent-sandbox-validation-matrix-readiness.md`:
- Land the three 🟡 integration tests called out in the checklist
  (behaviors 3, 4, 8).
- Mark all 9 behaviors ✅.
- Add a "Rollout flipped: 2026-XX-XX" line at the bottom.

D8. **Restart + smoke**:
- `vrooli scenario restart agent-manager workspace-sandbox swarm-manager web-console`
- Manual UI smoke: open `QuickRunDialog`, verify the default radio is
  Sandboxed, verify in-place opt-out copy.
- `curl agent-manager/metrics | grep agent_manager_runs_total` should return
  populated counters after a smoke spawn.

### Phase E — Protected-mode launch path

Closes `execute/protected-sandbox-agent-launch`.

E1. **Provider seam extension** —
`path:scenarios/agent-manager/api/internal/adapters/sandbox/interface.go`:
add `ExecProcess(ctx, sandboxID, ExecRequest) (ExecResult, error)` and
`LaunchProcess(ctx, sandboxID, LaunchRequest) (ProcessHandle, error)` and
`AttachInteractive(ctx, sandboxID, InteractiveOpts) (InteractiveSession, error)`
methods. Mirror the workspace-sandbox handler request/response shapes.

E2. **WorkspaceSandboxProvider impl** —
`adapters/sandbox/workspace_sandbox.go`: implement the three new methods
against `/api/v1/sandboxes/{id}/exec`, `/processes`, `/interactive` (the
last is a WebSocket; consider whether protected-mode launch needs PTY at
all or just streaming).

E3. **Runner adapter fork** — for each runner
(`adapters/runner/{claude_code,codex_runner,opencode_runner}.go`):
introduce a `launchInProtectedMode()` branch taken when
`cfg.SandboxConfig.Mode.Effective() == SandboxModeProtected`. The branch:
- Builds the same argv it currently passes to `exec.Command(...)`.
- Calls `provider.LaunchProcess(...)` (or `ExecProcess` for sync runs).
- Streams stdout/stderr through the same transcript consumer.
- Honors the same timeout/cancel semantics.

E4. **Tracking-mode preservation** — when `Mode == Tracking` (or
unspecified), the existing host-exec path is unchanged.

E5. **Capability matrix** — write
`path:scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md` documenting per-runner
support for: streaming output, interactive REPL, env injection, working
directory control, cleanup, network mode, timeout.

E6. **E2E test** —
`path:scenarios/agent-manager/api/internal/orchestration/protected_mode_e2e_test.go`:
spawn a fake runner via the workspace-sandbox `/exec` path against an
httptest.Server emulating workspace-sandbox; assert the same auditability
contract (provenance write, apply-at-run-end behavior) holds as in tracking
mode.

### Phase F — Protected-mode git + network guardrails

Closes `execute/protected-sandbox-git-and-network-guardrails`.

F1. **Workspace-sandbox `/exec` argv enforcement** —
`path:scenarios/workspace-sandbox/api/internal/handlers/process.go` (and `/exec`
handler if separate): when the request's parent sandbox has
`Behavior.Protected.GitAllowlist` non-empty AND `argv[0]` is `git` (or any
path resolving to `git`), reject any verb not in the allowlist.
Default allowlist: `status, diff, log, show, rev-parse`.
Response: 403 `{"error":"git_verb_blocked", "verb":"<v>", "message":"…"}`.

F2. **Bwrap network mode wiring** —
`path:scenarios/workspace-sandbox/api/internal/driver/bwrap.go:480-497` already
honors `NetworkAccess`. Verify protected-mode requests propagate
`SandboxConfig.NetworkMode` from the agent-manager side (Phase 3b added
`NetworkMode` to the proto SandboxConfig; ensure
`ProcessSpec.NetworkAccess` is sourced from it in the launch path, not from
the agent-profile-level field).

F3. **Denial UX in agent-manager** —
when the workspace-sandbox returns the 403 above, `WorkspaceSandboxProvider`
translates it into a typed `ErrToolBlocked{Verb, Reason}` and the runner
emits a `tool.blocked` event surfaced in the run timeline.

F4. **Documentation** —
`path:scenarios/workspace-sandbox/docs/EXECUTION_MODES.md` already documents
Vrooli-Aware mode (`localhost`); add a "Protected Git + Network" section
documenting the allowlist, the 403 shape, and the rationale (wrap-not-use:
agent uses GCT for mutations, not direct git).

F5. **Tests**:
- workspace-sandbox: `handlers/process_git_allowlist_test.go` — POST `/exec`
  with `argv=[git, commit]` returns 403; `argv=[git, status]` returns 200.
- workspace-sandbox: `handlers/process_network_mode_test.go` — `none` blocks
  outbound, `localhost` allows `127.0.0.1`, `full` allows arbitrary.
- agent-manager: `adapters/runner/protected_mode_git_block_test.go` — runner
  surfaces a typed error event when the sandbox returns 403.

### Phase G — Protected-mode policy enforcement surface

Closes `fix/protected-sandbox-policy-enforcement-surface`.

G1. **Policy interface slim** —
`path:scenarios/agent-manager/api/internal/policy/interface.go`:
- Already deleted `Decision.RequiresApproval` in Phase A.
- Decide for each remaining field whether it's enforced or advisory:
  - `RequiresSandbox` — enforced (router gates `RunMode`). Keep.
  - `EffectiveTimeout` — enforced (run executor cancels). Keep.
  - `EffectiveMaxFiles`, `EffectiveMaxSize` — currently advisory (sandbox
    `AcceptanceConfig` is the real gate). **Decision: delete from `Decision`**;
    callers must read directly from the resolved `SandboxConfig`.
  - `AppliedPolicies` — kept for audit.

G2. **Path enforcement via sandbox** — push `Profile.AllowedPaths` /
`DeniedPaths` into `SandboxConfig.Acceptance.Allow.PathGlobs` /
`Deny.PathGlobs` at config-resolve time
(`orchestration/service.go` resolve helpers). Runners stop receiving
`ALLOWED_PATHS`/`DENIED_PATHS` advisory env vars in protected mode (they're
already enforced at apply-at-run-end).

G3. **Resource limits** — for protected mode, plumb
`SandboxConfig.Resource.{MemoryMB,CPUTimeSec,MaxProcesses}` into
the workspace-sandbox `/processes` request (which already supports
`ResourceLimits` per `handlers/process.go`).

G4. **Tests**:
- `policy/interface_test.go` updated for slimmed Decision.
- `orchestration/policy_to_sandbox_test.go` — `Profile.AllowedPaths`
  appearing on the resolved `SandboxConfig.Acceptance.Allow.PathGlobs`.
- `orchestration/protected_mode_resource_limits_test.go` — limits
  forwarded to the LaunchProcess request.

G5. **Documentation** —
`path:scenarios/agent-manager/api/internal/policy/README.md` (new):
"agent-manager policy is now a thin layer that resolves
profile + per-run overrides into `SandboxConfig`. Real enforcement happens
at workspace-sandbox apply-at-run-end (acceptance) and in protected-mode
launch (path + resource limits + git/network)."

### Phase H — Final validation + handoff

H1. Full `go test ./...` on every touched scenario.
H2. `golangci-lint run ./...` clean on agent-manager + workspace-sandbox.
H3. `npx tsc --noEmit` clean on agent-manager + swarm-manager + web-console UIs.
H4. `vrooli scenario restart agent-manager workspace-sandbox swarm-manager web-console`.
H5. Live wire smokes: tracking-mode `apply-at-run-end`, protected-mode launch
    of `claude --version` (the smallest E2E proving the path works), git-block
    smoke, network-mode-`none` smoke.
H6. `swarm-manager backlog update --kind <k> --name <n> --data '{"status":"completed"}'`
    for all 6 closed items.
H7. `swarm-manager initiatives get` confirms 10/10 + 3/3.

---

## 9. Contract Decisions

These are the contract-level decisions **already locked** by the
auditability contract or the predecessor plan. They are not up for
re-litigation — listed here only so a resuming agent can sanity-check.

| # | Decision | Source |
|---|---|---|
| C1 | `SandboxMode` enum is `tracking` (default) and `protected`. No third value. | Auditability contract; `domain/types.go:217-231` |
| C2 | Apply-at-run-end is the **single decision point** for what gets applied. Sandbox-side `CanAutoApprove` is dead. | Predecessor plan Phase 3a/3b |
| C3 | Per-run levers live on `SandboxConfig`: `Mode, ManualReview, AutoApply, ApplyOnFailure, NetworkMode, NoLock` + `Acceptance.{Allow,Deny}`. No `RequiresApproval`, no `AutoApprove`, no `AutoReject`. | Predecessor plan |
| C4 | Sandboxed default at every spawn surface. In-place is an explicit operator opt-out. | This plan §7 |
| C5 | Schema-version package versioned `1.0.0` and shared with GCT-pending-AI-provenance. | Backlog item description |
| C6 | Protected-mode git allowlist: `status, diff, log, show, rev-parse`. Block: `branch, commit, checkout, reset, rebase, merge, push, pull, clean`. Wrap-not-use: GCT owns mutating git. | Backlog item description |
| C7 | Cron + direct-CLI spawn-surface flips are out of scope; tracked separately after this plan. | Default-rollout backlog item |

---

## 10. Testing Plan

### Unit / package tests (run between every phase)

| Scenario | Command | Phases that touch it |
|---|---|---|
| agent-manager | `cd scenarios/agent-manager/api && go test ./...` | A, C, D, E, F, G, H |
| workspace-sandbox | `cd scenarios/workspace-sandbox/api && go test ./...` | A, B, F, H |
| swarm-manager | `cd scenarios/swarm-manager/api && go test ./...` | A, C, D, H |
| ecosystem-manager | `cd scenarios/ecosystem-manager && go test ./...` | A, H |
| app-issue-tracker | `cd scenarios/app-issue-tracker && go test ./...` | A, H |
| scenario-auditor | `cd scenarios/scenario-auditor && go test ./...` | A, H |
| test-genie | `cd scenarios/test-genie && go test ./...` | A, H |
| sandbox-provenance pkg | `cd packages/sandbox-provenance/go && go test ./...` | B, H |

### Lint / format (between every phase)

```bash
cd scenarios/agent-manager/api      && golangci-lint run ./...
cd scenarios/workspace-sandbox/api  && golangci-lint run ./...
gofumpt -d <touched-files>          # diff before apply
```

### TypeScript / UI

```bash
cd scenarios/agent-manager/ui  && npx tsc --noEmit
cd scenarios/swarm-manager/ui  && npx tsc --noEmit
cd scenarios/web-console/ui    && npx tsc --noEmit
```

### Live wire smokes (after every restart)

```bash
vrooli scenario restart agent-manager workspace-sandbox swarm-manager web-console

# basic health
curl -fsS localhost:<agent-manager-port>/health
curl -fsS localhost:<workspace-sandbox-port>/health

# tracking-mode apply-at-run-end (existing smoke from Phase 3b)
vrooli scenario logs agent-manager --tail 50

# protected-mode launch smoke (added in Phase E)
agent-manager run create --runner=claude --mode=protected --task=… # shape TBD
agent-manager run get <id> | jq '.events[] | select(.type=="run.completed")'

# git-block smoke (added in Phase F)
workspace-sandbox exec <sb> -- git status   # 200
workspace-sandbox exec <sb> -- git commit   # 403, structured

# adoption metrics (Phase D)
curl -fsS localhost:<agent-manager-port>/metrics | grep agent_manager_runs_total
```

### Integration tests added by this plan

| Phase | Test | Purpose |
|---|---|---|
| B | `apply_at_run_end_persistence_test.go` | runOutcome/state/conversationId/costUsd persisted |
| C | `swarm-manager service_test.go` extension | ConversationID/ParentRunID populated on every spawn |
| D | rollout-readiness 🟡 → ✅ tests (3 sandbox behaviors) | Validation matrix closes out |
| E | `protected_mode_e2e_test.go` | Auditability contract holds in protected mode |
| F | `process_git_allowlist_test.go`, `process_network_mode_test.go`, `protected_mode_git_block_test.go` | Real guardrails |
| G | `policy_to_sandbox_test.go`, `protected_mode_resource_limits_test.go` | Policy-to-sandbox handoff is real |

---

## 11. Rollout / Validation Checklist

Each phase ends with this gate before proceeding to the next:

- [ ] Code compiles on every touched scenario (`go build ./...`).
- [ ] `go vet ./...` clean.
- [ ] `golangci-lint run ./...` clean (no new findings; gofumpt diffs applied).
- [ ] `go test ./...` green on every touched scenario.
- [ ] `npx tsc --noEmit` green on every touched UI.
- [ ] `vrooli scenario restart <touched-scenarios>` succeeds.
- [ ] `/health` returns ok on every touched scenario.
- [ ] Live wire smoke for the phase's deliverable returns the expected shape.
- [ ] `git grep` checks in §7 row 2 still pass (no regressions of legacy refs).

After the final phase:

- [ ] `swarm-manager initiatives get --name agent-sandbox-audit-foundation`
      reports 10/10 completed.
- [ ] `swarm-manager initiatives get --name protected-agent-sandboxing`
      reports 3/3 completed.
- [ ] `path:docs/plans/agent-sandbox-validation-matrix-readiness.md` shows all 9
      behaviors ✅ and a rollout-flipped date.
- [ ] No references to `RequiresApproval`, `AutoApprove`, `AutoReject`,
      `DisableAutoApproveIfEmpty` remain outside (a) per-tool inbox usage
      and (b) reserved proto markers.
- [ ] Comment-trail audit: `git grep 'removed in Phase'` returns zero hits.

---

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Production DB has rows where `requires_approval=1` and dropping the column loses operator intent. | Low (Phase 3b set writes to `false`; live runs don't depend on it). | Med (silent loss of "must review" intent). | Phase A2 migration first runs `UPDATE agent_profiles SET sandbox_config = jsonb_set(...) WHERE requires_approval = 1` to set `manualReview=true` on the JSON column, **then** drops the column. Idempotent. |
| Removing `policy.CanAutoApprove` breaks an unknown caller in another scenario. | Low (`git grep` shows none). | Low (compile error, easy to spot). | Run cross-scenario builds in Phase A9; treat any failure as a missing inventory entry, fix in same PR. |
| Protected-mode launch path adds latency that breaks existing timeout assumptions in tests. | Med | Med | Phase E5's capability matrix includes per-runner overhead measurements; bump default timeouts in `protected_mode_e2e_test.go` if needed (greenfield: change defaults in domain layer, not add per-test overrides). |
| Swarm-manager `agentRequiresApproval` deletion surprises operators relying on the global toggle. | Med | Low (the toggle was already inert for runs spawned post-Phase 3b). | Release-note the deletion in the swarm-manager UI changelog; if user wants the global, take Phase A5's "rename to `agentManualReview` and wire through" alternative — but default is delete. |
| Schema-version package version drift between this initiative and GCT initiative. | Med | High (two readers disagree on field shapes). | Phase B2's `COORDINATION.md` pins the contract; bump to `1.1.0` only via a coordination commit touching both branches. |
| Default-flip causes regressions in one of the spawn surfaces still on the legacy path. | Low | Med | Phase D4 explicit fallback emits a `WARN` event so any incompatible scenario is observable, not silent. |
| The agent-manager `sandbox_config` JSONB migration touches a large table. | Low | Low | The table is per-installation small (<10k rows in any current deployment); do it inline in the migration. |

---

## 13. Non-Goals / Prohibited Patterns

- **Do not** add a "compatibility shim" or "transition adapter" of any kind.
  If you find yourself needing one, the phase ordering is wrong — fix the
  ordering, not the code.
- **Do not** preserve `// removed in Phase X` comments. Delete them.
- **Do not** preserve `_unused` aliases or `// kept for compatibility` exports.
- **Do not** add a `legacy_` table or column name. The migration drops the
  column; that is the entire compatibility story.
- **Do not** redesign the contract — `mode`, `manualReview`, `autoApply`,
  `applyOnFailure`, `networkMode`, `noLock` are locked by the auditability
  research finding. Wire what's there.
- **Do not** flip cron or direct-CLI spawn surfaces — those are tracked as
  separate items per the default-rollout backlog item description.
- **Do not** touch tool-level `inbox.ToolMeta.RequiresApproval` (different
  concern, not legacy of this initiative).
- **Do not** introduce a sandbox-side approval policy. The new contract makes
  agent-manager's `apply-at-run-end` the sole apply decision; sandbox-side
  `CanAutoApprove` was dead before, and this plan deletes it.
- **Do not** mock the database in integration tests (project rule per
  `feedback_testing.md` lineage).
- **Do not** commit, push, or revert anything (project rule:
  `feedback_no_git_mutations.md`).

---

## 14. Definition of Done

All of the following must be true for this plan to be considered complete:

1. `swarm-manager initiatives get --name agent-sandbox-audit-foundation`
   reports 10/10 completed.
2. `swarm-manager initiatives get --name protected-agent-sandboxing`
   reports 3/3 completed.
3. `git grep -nE 'RequiresApproval|AutoApprove|AutoReject|DisableAutoApproveIfEmpty'`
   inside the seven affected scenarios returns **only** per-tool
   `inbox.ToolMeta.RequiresApproval` references in `toolregistry/` directories
   and `reserved` markers in `path:packages/proto/schemas/agent-manager/`.
4. `git grep 'removed in Phase'` returns **zero** hits.
5. `requires_approval` is gone from `schema.sql`, `repository.go`,
   `connection.go`, and any production database after migration runs.
6. `golangci-lint run ./...` is clean on agent-manager + workspace-sandbox.
7. `go test ./...` is green on all seven affected scenarios + the new
   `path:packages/sandbox-provenance/go` package.
8. `npx tsc --noEmit` is green on agent-manager UI, swarm-manager UI, and
   web-console UI.
9. `agent-manager`, `workspace-sandbox`, `swarm-manager`, `web-console` have
   been restarted via `vrooli scenario restart` and report `/health` ok.
10. `QuickRunDialog`'s default `runMode` is `RunMode.SANDBOXED`; in-place is
    explicitly labelled as the operator escape hatch.
11. `swarm-manager stats sandbox-adoption` returns populated counters after
    a smoke spawn cycle.
12. Live wire smokes pass for: tracking-mode apply-at-run-end (existing),
    protected-mode launch of a trivial command, git-verb block, network-mode
    `none` block.
13. `path:docs/plans/agent-sandbox-validation-matrix-readiness.md` shows all 9
    behaviors ✅ and a "Rollout flipped: 2026-XX-XX" line.
14. No backwards-compatibility shims, dual-name fields, `// removed` comments,
    or `_unused` aliases were introduced.
15. The `path:packages/sandbox-provenance` package exists, is versioned 1.0.0,
    has a `COORDINATION.md` for the GCT branch, and is consumed by
    workspace-sandbox.
16. The user has reviewed the resulting diffs and approved.

---

## Appendix A — Resume hints for a fresh agent

If you arrived here without prior context:

1. Read §2's required reading commands top to bottom.
2. Run `swarm-manager initiatives get --name agent-sandbox-audit-foundation`
   and `--name protected-agent-sandboxing` to see how much is already done.
3. Run the §10 unit-test suite on agent-manager + workspace-sandbox to
   confirm the baseline is green.
4. Run `git grep -nE 'RequiresApproval|AutoApprove|AutoReject|DisableAutoApproveIfEmpty'`
   in the seven affected scenarios — match the output against §4.2 and §4.3
   to see what cleanup is still pending.
5. Pick up at the earliest unfinished phase in §8.
6. Do **not** run any `git` mutation commands — the user owns all git
   operations (`feedback_no_git_mutations.md`).

## Appendix B — Files most likely to be edited

```
scenarios/agent-manager/api/internal/
  policy/interface.go
  policy/interface_test.go
  database/connection.go
  database/repository.go
  database/schema.sql
  domain/types.go
  domain/types_test.go
  orchestration/service.go
  orchestration/run_executor.go
  orchestration/investigation.go
  orchestration/sandbox_config_test.go
  handlers/handlers.go
  adapters/sandbox/interface.go
  adapters/sandbox/workspace_sandbox.go
  adapters/runner/claude_code.go
  adapters/runner/codex_runner.go
  adapters/runner/opencode_runner.go
  protoconv/{convert,entities}.go

scenarios/agent-manager/ui/src/
  types.ts
  hooks/useApi.ts
  hooks/useTasksRunDialogState.ts
  components/QuickRunDialog.tsx
  components/ProfileDetail.tsx
  components/RunDetail.tsx
  pages/{ProfilesPage,TasksPage}.tsx

scenarios/workspace-sandbox/api/internal/
  types/types.go
  types/auditability_contract_test.go
  policy/policy.go              (delete or slim)
  policy/approval.go            (delete)
  policy/policy_test.go         (delete relevant cases)
  config/config.go
  config/config_test.go
  logging/logging_test.go
  sandbox/manual_review_ttl_test.go
  sandbox/apply_at_run_end_persistence_test.go  (NEW)
  handlers/process.go
  handlers/process_git_allowlist_test.go        (NEW)
  handlers/process_network_mode_test.go         (NEW)
  database/schema.sql
  database/connection.go        (provenance migration)
  driver/bwrap.go               (verify NetworkMode wiring)

scenarios/swarm-manager/api/internal/agentmanager/service.go
scenarios/swarm-manager/api/internal/settings/{adapters,adapters_test}.go
scenarios/swarm-manager/cli/cmd_settings.go
scenarios/swarm-manager/cli/cmd_stats.go              (sandbox-adoption subcommand)
scenarios/swarm-manager/ui/src/services/settings-service.ts
scenarios/swarm-manager/ui/src/services/proto/settings-contracts.ts
scenarios/swarm-manager/ui/src/types/settings.ts
scenarios/swarm-manager/ui/src/components/settings/ExecutionTab.tsx

scenarios/ecosystem-manager/api/pkg/agentmanager/service.go
scenarios/app-issue-tracker/api/internal/agentmanager/service.go

packages/proto/schemas/agent-manager/v1/domain/{profile,types}.proto
packages/sandbox-provenance/go/schema.go             (NEW)
packages/sandbox-provenance/go/schema_test.go        (NEW)
packages/sandbox-provenance/ts/schema.ts             (NEW)
packages/sandbox-provenance/COORDINATION.md          (NEW)

docs/plans/agent-sandbox-validation-matrix-readiness.md
scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md (NEW)
scenarios/agent-manager/api/internal/policy/README.md  (NEW)
scenarios/workspace-sandbox/docs/EXECUTION_MODES.md
```
