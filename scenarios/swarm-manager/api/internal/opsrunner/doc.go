// Package opsrunner is the single runtime chokepoint of the declarative
// agent-operations layer: it turns a typed domain operation request — (target
// ref, operation id+version, caller inputs) — into one reproducible operating-
// mode execution and a typed result, and correlates it in a durable domain
// workflow instance.
//
// The runner is deliberately target- and mode-agnostic. It never branches on a
// named target kind (backlog-item/initiative), a named mode, or a named phase;
// those live in data (operation contracts, bindings, policies) and behind two
// injected seams — a ModePreparer (compiles+pins the bound mode and validates
// inputs) and an ExecutionDriver (drives the run: a deterministic simulation
// seam for tests and offline reproduction, or a live agent-spawning driver).
// Every decision that determined a run is pinned into an immutable
// agentops.ExecutionProvenance record (mode revision + digests + binding + caller
// input digest + policy revision), so a historical execution stays resolvable
// after the source definitions change, and a digest mismatch is a typed error
// rather than a silent divergence.
//
// Responsibilities, all funneled through Invoke:
//   - compatibility + input validation (agentops contracts),
//   - layered binding resolution with explicit provenance and fail-closed
//     precedence (catalog system defaults + domain overrides),
//   - immutable provenance + digest pinning for reproducibility,
//   - durable, domain-owned workflow persistence with optimistic concurrency and
//     idempotency keys,
//   - a closed, deterministic domain-action dispatcher + transition evaluator,
//   - restart-safe scheduled intents,
//   - run-owner attribution for the evidence ledger.
package opsrunner
