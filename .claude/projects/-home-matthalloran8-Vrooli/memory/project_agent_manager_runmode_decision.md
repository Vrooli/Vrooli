---
name: Agent-manager RunMode decision boundary
description: SandboxConfig.Mode is the single source of truth for sandboxed-vs-in-place; RequiresSandbox bool is gone.
type: project
---
# Agent-manager RunMode decision (post-Phase-1, 2026-04-29)

**Single source of truth:** `domain.SandboxConfig.Mode` — every value except `SandboxModeOff` produces `RunModeSandboxed`. Use `domain.DeriveRunMode(*SandboxConfig)` to translate; do not reinvent the rule elsewhere.

**Why it changed:** The old `RequiresSandbox bool` field on `RunConfig`, `AgentProfile`, and `policy.Decision` had a Go zero-value pit — a profile or request that forgot to set the bool silently dropped the run to in-place, even when policy and profile defaults wanted sandboxed. The bypass was responsible for codex's `cwd` ending up at `/home/matthalloran8/Vrooli` (the canonical repo) on protected-mode runs that should have stayed inside `workspace-sandbox`.

**How to apply:**
- New `SandboxModeOff` is the explicit "no sandbox" choice. Profiles/runs that need in-place execution must set `SandboxConfig.Mode = SandboxModeOff` (proto: `SANDBOX_MODE_OFF`). Leaving SandboxConfig nil is also treated as in-place by `DeriveRunMode`, but spawn surfaces should clone `DefaultSandboxConfig()` so the safe default is sandboxed.
- Policies declare a minimum sandbox strictness via `policy.Decision.RequiredSandboxMode SandboxMode` (zero = no requirement). The orchestrator rejects the run when `SandboxConfig.Mode < required`. Use `SandboxMode.AtLeast(required)` for comparison; the rank order is `Off (0) < Tracking (1) < Protected (2)`.
- The CLI exposes `--sandbox-mode off|tracking|protected` (replacing the old `--sandbox` boolean) on `agent-manager profile create|update|ensure`.
- The UI `ProfileFormData.sandboxMode` and `RunFormData.sandboxMode` carry the form-level string; `useApi.ts` maps it to `SandboxConfig{Mode: …}` on the proto request.
- Downstream scenario clients (test-genie, system-monitor, scenario-to-cloud, scenario-to-desktop, swarm-manager, app-issue-tracker, scenario-auditor, knowledge-observatory, prompt-manager, swarm-manager) expose their own `SandboxMode` enum field on their `ProfileConfig` and let `agent-manager.resolveSandboxConfig` backfill the rest of the contract defaults.

**Don't do this:**
- Reintroduce a `RequiresSandbox bool` anywhere — that's the bug.
- Compute `runMode` from anything other than `DeriveRunMode + optional req.RunMode/ForceInPlace`. `service.go:CreateRun` is the only legitimate composer.
- Persist `requires_sandbox` in any new DB schema or proto message — the proto field IDs are reserved (`agent-manager/v1/domain/profile.proto`).
