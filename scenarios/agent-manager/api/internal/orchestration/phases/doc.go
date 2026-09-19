// Package phases contains the per-phase logic for run execution, extracted
// from the orchestration.RunExecutor god-object as part of the agent-manager
// refactor (Phase 6 of the runner+executor consolidation).
//
// Each phase exports a single function with an explicit input struct (no
// shared *RunExecutor receiver). Phase ordering is owned by run_executor.go's
// Execute() coordinator and nowhere else; this package contains no
// orchestration logic, only the work each phase does in isolation.
//
// Files:
//   - deps.go       — shared dependency bundle (events, runs, broadcaster, gate, levers)
//   - emitters.go   — event emission helpers used by every phase, routed through the Gate
//   - env.go        — sandbox + identity env-var construction
//   - setup.go      — workspace creation (sandbox + in-place)
//   - acquire.go    — runner selection + fallback chain walk
//   - execute.go    — runner.Execute wrapper + model fallback chain walk
//   - validate.go   — silent-launch failure detection
//   - result.go     — outcome classification + handler dispatch
//   - finalize.go   — apply-at-run-end + sandbox teardown (deferred terminal seam)
//   - failure.go    — FailWithError, HandleContextError, CleanupOnFailure helpers
//   - checkpoint.go — phase-ladder advancement + checkpoint persistence
//   - heartbeat.go  — run heartbeat goroutine body
//   - identity.go   — identity-token generation
//
// DOC: scenarios/agent-manager/docs/concepts/ARCHITECTURE.md (folder structure,
// invariants), scenarios/agent-manager/docs/internal/SEAMS.md (testability
// boundaries), scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md
// (per-phase cadences and timeouts).
package phases
