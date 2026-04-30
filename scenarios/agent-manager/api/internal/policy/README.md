# agent-manager `policy` package

Slim policy seam left after the agent-sandbox audit foundation cleanup
(Phase G of `docs/plans/agent-sandbox-completion-and-protected-mode-implementation-plan.md`).

## What this package owns

- The `Evaluator` interface — the seam for policy decisions.
- The `Decision` struct — what the evaluator returns. Fields:
  - `Allowed`, `DenialReason`, `DenialPolicy` — go/no-go.
  - `RequiredSandboxMode` — minimum sandbox strictness the run must
    satisfy (`Off < Tracking < Protected`); zero-value
    `SandboxModeUnspecified` means "no minimum". The orchestrator
    rejects the run if the resolved `SandboxConfig.Mode` is below this
    minimum.
  - `EffectiveTimeout` — the run executor enforces.
  - `AppliedPolicies` — audit trail of which policy contributed.
- `EvaluateRequest`, `ConcurrencyDecision`, `ApprovalDecision` —
  request/response shapes for the evaluator surface.

## What this package does **not** own (deliberately)

- **`RequiresApproval`** — removed in Phase 3b of the auditability cutover.
  Operator-gated apply lives on `domain.SandboxConfig.ManualReview`, decided
  per-run in `orchestration.resolveSandboxConfig`.
- **`EffectiveMaxFiles` / `EffectiveMaxSize`** — removed in Phase G. These
  were stored on `Decision` but never enforced anywhere. Real per-run file
  caps live on workspace-sandbox `Behavior.Acceptance` and are enforced at
  apply-at-run-end.
- **`AllowedPaths` / `DeniedPaths` enforcement** — these still live on
  `domain.AgentProfile` and `RunConfig` (advisory/visibility), but the
  load-bearing enforcement happens through `SandboxConfig.Acceptance.Allow.PathGlobs`
  and `Acceptance.Deny.PathGlobs`. Phase G's `resolveSandboxConfig`
  pushes profile/req paths into the acceptance layer at run-creation time.
- **Approval policy** (auto-approve thresholds, etc.) — used to live on the
  workspace-sandbox `policy.ApprovalPolicy` interface; deleted in Phase A.
  The single decision point is now agent-manager's `applyAtRunEnd`, which
  consults `SandboxConfig.{ManualReview, AutoApply, ApplyOnFailure}`.

## Design intent

agent-manager's policy layer is now a thin resolver. Real enforcement happens
at the boundary that owns the resource:

| Concern | Enforced by | Surface |
|---|---|---|
| Sandbox required? | agent-manager | `Decision.RequiredSandboxMode` enforces a minimum on `SandboxConfig.Mode`; `domain.DeriveRunMode` derives the binary `RunMode` from the resolved Mode |
| Timeout | agent-manager | run executor cancels |
| Apply gate (manual vs auto) | agent-manager | `applyAtRunEnd` reads `SandboxConfig` |
| Per-file allow/deny paths | workspace-sandbox | `apply-at-run-end` filters by `Behavior.Acceptance.{Allow,Deny}.PathGlobs` |
| Network mode (none/localhost/full) | workspace-sandbox driver | `bwrap.go` network namespace flags from `SandboxConfig.NetworkMode` |
| Direct git verb allowlist (protected mode) | workspace-sandbox `/exec` | argv inspection (Phase F) |
| Memory / CPU / process limits (protected mode) | workspace-sandbox `/processes` | `ResourceLimits` forwarded from agent-manager |

If you find yourself adding a "advisory" field to `Decision` that the runtime
cannot enforce, stop and reconsider — push the enforcement to the layer that
owns the resource instead.
