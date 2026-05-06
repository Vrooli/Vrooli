# Agent Session Sandbox Environment Implementation Plan

## Purpose

Implement a clean, long-term fix for agent session execution correctness across `agent-manager`, `workspace-sandbox`, and `swarm-manager`.

This plan is based on the investigation of agent-manager run `0c482222-49b0-4f16-8ddc-c0461c8d3790` and swarm-manager session `sess_84f07e2fa5fe31f7`. The observed failures are:

- Agent-manager durable run events duplicate the final Claude assistant response.
- Sandboxed agent processes do not receive a Vrooli-aware tool environment, so project-level CLIs and sandbox host requirement checks fail.
- Swarm-manager session storage does not duplicate the assistant response, which means the duplication is currently isolated to agent-manager run events and views that replay those events.
- Scenario-local swarm-manager CLI binaries can be stale relative to the canonical installed CLI.

The requested direction is greenfield: no compatibility shims, no legacy branches, no dead code, and no UI-only masking of backend correctness issues.

## Required Reading

Before implementing, load these runtime skills:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read scientific-debugging
```

Also read:

- `docs/scenario-qa/investigation-techniques/scientific-debugging.md`
- `scenarios/agent-manager/api/internal/adapters/runner/codecs/claude.go`
- `scenarios/agent-manager/api/internal/adapters/runner/codecs/golden_test.go`
- `scenarios/agent-manager/api/internal/adapters/runner/transcript_consumer.go`
- `scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher.go`
- `scenarios/workspace-sandbox/api/internal/config/profiles.go`
- `scenarios/workspace-sandbox/api/internal/driver/exec/config.go`
- `scenarios/workspace-sandbox/api/internal/driver/exec/args.go`
- `scenarios/workspace-sandbox/api/internal/driver/exec/run.go`
- `scenarios/workspace-sandbox/api/internal/handlers/process_start.go`
- `scenarios/swarm-manager/api/internal/agentsessions/service.go`
- `scenarios/swarm-manager/cli/domains/sessions/register.go`

## Problem Statement

Agent-manager run `0c482222-49b0-4f16-8ddc-c0461c8d3790` completed successfully from the runner's perspective, but the content shows the agent could not use the expected local Vrooli tooling inside its sandbox. The event list also contains the same final assistant message twice.

Observed event sequence:

- `swarm-manager overview --format markdown` failed with `swarm-manager: command not found`.
- A local scenario CLI was found under `/workspace/scenarios/swarm-manager/cli`, but `swarm-manager sessions` was unavailable in that binary.
- Starting swarm-manager through the local CLI failed because `vrooli` was not on `PATH`.
- Manually exporting `/workspace` and `/home/matthalloran8/go/bin` allowed `vrooli scenario status swarm-manager` to run.
- Starting swarm-manager then failed because workspace-sandbox reported missing host requirements: `buf`, `protoc`, `protoc-gen-connect-go`, `protoc-gen-es`, and `protoc-gen-go`.
- Agent-manager events included two identical final assistant messages at adjacent event sequences.
- Swarm-manager session storage contained only one assistant message in `messages.jsonl`, and `swarm-manager sessions get --id sess_84f07e2fa5fe31f7 --json` returned one assistant message.

The root problem is not a single UI bug. It is a contract gap between the agent runtime, durable transcript decoding, and the sandboxed process environment.

## Scope

In scope:

- Make Claude durable transcript decoding stateful enough to avoid duplicate final assistant messages while preserving fallback result text when no assistant message event exists.
- Make workspace-sandbox's Vrooli-aware profile provide a deterministic, audited, useful tool environment for Vrooli agents.
- Ensure agent-manager sandbox launches pass and preserve the environment contract needed by project-level CLIs.
- Validate swarm-manager session refresh and display behavior against corrected agent-manager events.
- Decide how agents should locate canonical CLIs and eliminate stale local-binary ambiguity from the supported path.
- Add regression tests and one end-to-end validation path.

Out of scope:

- Preserving duplicated event behavior.
- Adding UI deduplication as the primary fix.
- Teaching agent prompts to export ad hoc `PATH` values.
- Installing missing dependencies as part of this fix.
- Bypassing scenario lifecycle commands.
- Supporting direct scenario API execution.
- Maintaining legacy CLI discovery fallbacks that contradict the canonical Vrooli environment.

## Technical Context

### Agent-Manager Transcript Decoding

`scenarios/agent-manager/api/internal/adapters/runner/codecs/claude.go` currently has separate live stream and durable transcript paths. The live path uses a shared Claude codec state. The durable transcript path can call `ParseTranscriptLine` per transcript line.

`ParseTranscriptLine` currently creates a fresh `claudeState` for each line. That means a later `result` transcript line cannot know that an earlier assistant message line already emitted the same final content. The parser can therefore emit one assistant message for the assistant message line and another synthesized assistant message from the final `result` line.

This explains why agent-manager run events duplicate the final assistant response while swarm-manager's session store does not. Swarm-manager session refresh writes a single assistant summary and is not replaying raw duplicated assistant events into `messages.jsonl`.

### Workspace-Sandbox Environment

`scenarios/workspace-sandbox/api/internal/handlers/process_start.go` copies request env values into exec config. `scenarios/workspace-sandbox/api/internal/driver/exec/config.go` then applies the selected isolation profile.

The `vrooli-aware` profile currently sets `PATH` to:

```text
$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin
```

Profile env application overwrites existing keys, so request-provided `PATH` entries can be lost. The current default excludes canonical Vrooli development paths such as:

- `$HOME/.vrooli/bin`
- `$HOME/go/bin`
- The repo root when needed for local wrappers

The host has relevant tools in those locations. The sandbox should expose the correct audited environment without requiring an agent to discover and export those paths manually.

### Swarm-Manager Sessions

`scenarios/swarm-manager/api/internal/agentsessions/service.go` refreshes agent runs into session messages and already avoids appending duplicate assistant summaries across refreshes. The session backing file for `sess_84f07e2fa5fe31f7` contains exactly one assistant message.

This should remain a session-level idempotence guarantee, but it must not be used as a substitute for fixing duplicated agent-manager run events.

### CLI Discovery

The canonical installed CLI under `.vrooli/bin` supports `swarm-manager sessions`. The scenario-local CLI binary observed during the failed run did not. Agents need a single supported command discovery contract.

The greenfield contract should be: sandboxed agents use canonical installed Vrooli CLIs from the Vrooli-aware environment. Scenario-local binaries are development artifacts unless the lifecycle system explicitly builds and validates them.

## Target End State

- Agent-manager run events contain exactly one final assistant message for a Claude run transcript that has both an assistant content event and a final result line.
- Claude result text is still preserved when a transcript contains a result line but no assistant content event.
- Live tail consumption and final durable drain share enough codec state that the final drain cannot re-emit content already seen live.
- A sandboxed Vrooli agent can run these commands without manual environment mutation:
  - `vrooli`
  - `swarm-manager`
  - `buf`
  - `protoc`
  - `protoc-gen-go`
  - `protoc-gen-connect-go`
  - `protoc-gen-es`
- Workspace-sandbox remains auditable: process env is deterministic, minimal, and built by first-class code rather than inherited shell accidents.
- Swarm-manager session detail views and CLI session reads agree with agent-manager's corrected event stream.
- Stale scenario-local CLI binaries are not part of the supported agent execution path.

## Implementation Strategy

### Phase 0: Add Failing Reproduction Tests

Write tests before modifying behavior.

Agent-manager tests should reproduce:

- A Claude transcript containing an assistant message line followed by a `result` line with the same final text emits one assistant message event.
- A Claude transcript containing a `result` line with final text but no assistant message line emits one assistant message event.
- A live-tail phase that sees the assistant content followed by a final-drain phase that sees the result line does not emit the assistant content again.

Workspace-sandbox tests should reproduce:

- Applying `vrooli-aware` no longer drops required Vrooli CLI/toolchain paths.
- Request env and profile env are merged through an explicit contract, not last-write-wins clobbering.
- Existing profile bind discipline is preserved: do not add ad hoc per-subpath binds for `$HOME/.local/bin`, `$HOME/go/bin`, or `.vrooli/bin`; rely on the existing home overlay and explicit env.

Swarm-manager tests should reproduce:

- Refreshing the same completed agent run more than once appends at most one assistant summary.
- Session reads return a single assistant message when agent-manager exposes a single final assistant event.
- CLI command registration includes `sessions` in the canonical build.

### Phase 1: Make Transcript Decoding Stateful

Replace the stateless durable transcript decoding seam with a stateful transcript decoder contract.

Recommended design:

- Use one codec state across a whole transcript consume operation.
- Use the same state across live tail and final drain when both happen in one runner lifecycle.
- Let Claude transcript parsing synthesize an assistant message from `result` only when the codec state has not already seen assistant content for that run.
- Keep metrics, result status, and terminal metadata emission independent from assistant content synthesis.

This is cleaner than special-casing a one-line dedupe filter because it encodes the real invariant: transcript decoding is a stream, not a set of unrelated lines.

Implementation notes:

- Update the runner codec interface to make transcript parsing state explicit. This can be a greenfield interface change because this is internal scenario code.
- Update all runner codecs at the same seam so the interface is coherent.
- Update golden transcript tests to assert semantic event content and counts, not only event-type histograms.
- Avoid adding post-processing dedupe in persistence or UI layers.

### Phase 2: Define the Vrooli-Aware Sandbox Environment Contract

Create a single environment composition function in workspace-sandbox's exec/config layer.

The contract should:

- Expand `$HOME` deterministically.
- Build `PATH` from allowlisted Vrooli development locations and standard system locations.
- Preserve safe request-provided path additions when the caller intentionally supplies them.
- Deduplicate path entries while preserving order.
- Keep profile env authoritative for sandbox policy keys, but avoid clobbering useful caller env without an explicit rule.

Recommended default Vrooli-aware path:

```text
$HOME/.vrooli/bin:$HOME/go/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin
```

If the repo root wrapper remains a supported entry point, include `/workspace` deliberately and test it. Otherwise, keep `/workspace` out of `PATH` and require canonical CLIs from `$HOME/.vrooli/bin`.

Implementation notes:

- Put env composition in workspace-sandbox, not agent-manager, so every Vrooli-aware sandbox user benefits from the same contract.
- Keep agent-manager responsible only for selecting the correct profile and passing run identity/audit metadata.
- Add clear code comments around why the Vrooli-aware profile includes these paths.
- Fail explicitly when a required host tool is unavailable instead of letting agents discover missing tools through ambiguous command failures.

### Phase 3: Tighten Agent-Manager Sandbox Launch

Update agent-manager sandbox launcher tests to assert:

- The launcher selects the Vrooli-aware profile for sandboxed Vrooli runs.
- Identity env such as run ID, sandbox ID, workspace path, and session metadata is passed through.
- The command is translated into namespace paths correctly.
- The environment sent to workspace-sandbox does not depend on a developer's interactive shell state.

Do not hardcode host-specific paths in agent-manager. Host path policy belongs in workspace-sandbox.

### Phase 4: Normalize CLI Expectations

Make the canonical command path explicit:

- Agents should call `vrooli` and scenario CLIs from the installed Vrooli environment.
- Scenario-local CLI binaries should not be used by agents unless lifecycle tooling built them and validated they match the source command registry.

Recommended approach:

- Document this contract in the relevant internal runner/session docs.
- Add a small validation test or lifecycle check that the canonical swarm-manager CLI exposes `sessions`.
- If scenario-local binaries are retained for development, ensure lifecycle build commands refresh them. Do not add fallback logic that silently chooses stale binaries.

### Phase 5: Validate Swarm-Manager Session Behavior

Keep session idempotence as a defensive property, but do not hide agent-manager bugs in swarm-manager.

Add or strengthen tests around:

- `Refresh` appending assistant summaries only once.
- Session detail API returning the expected message list after a completed run.
- UI rendering one assistant message when the API returns one assistant message.
- UI not implementing content-based dedupe for backend event corruption unless there is a separate product requirement.

## Contract Decisions

- Durable transcript decoding is stateful.
- Claude `result` lines may synthesize assistant content only as a fallback when no assistant content was already emitted in the current transcript state.
- Live tail and final drain are one logical stream and must share codec state when run in the same lifecycle.
- Workspace-sandbox owns the Vrooli-aware environment contract.
- Agent-manager owns run identity, command selection, and sandbox profile selection, not low-level host path policy.
- Swarm-manager sessions remain idempotent, but agent-manager run events must be correct at the source.
- Canonical installed CLIs are the supported command surface for sandboxed agents.

## Testing Plan

### Agent-Manager Unit Tests

Add tests under `scenarios/agent-manager/api/internal/adapters/runner/codecs` and runner transcript consumption tests as needed:

- `TestClaudeTranscriptDoesNotDuplicateFinalAssistant`
- `TestClaudeTranscriptResultSynthesizesAssistantWhenNoAssistantEventExists`
- `TestClaudeLiveTailAndFinalDrainShareAssistantState`

Run:

```bash
cd scenarios/agent-manager/api && go test ./internal/adapters/runner/... ./internal/adapters/sandbox/... -timeout 300s
```

### Workspace-Sandbox Unit Tests

Add tests under:

- `scenarios/workspace-sandbox/api/internal/config`
- `scenarios/workspace-sandbox/api/internal/driver/exec`
- `scenarios/workspace-sandbox/api/internal/handlers`

Test:

- Vrooli-aware `PATH` contains `$HOME/.vrooli/bin`, `$HOME/go/bin`, `$HOME/.local/bin`, `/usr/local/bin`, `/usr/bin`, and `/bin`.
- Path merge deduplicates entries and preserves deterministic order.
- Profile env does not accidentally clobber request env when the merge contract says to preserve it.
- Existing no-ad-hoc-home-subpath-bind tests continue passing.

Run:

```bash
cd scenarios/workspace-sandbox/api && go test ./internal/config ./internal/driver/exec ./internal/handlers -timeout 300s
```

### Swarm-Manager Tests

Add or strengthen tests under:

- `scenarios/swarm-manager/api/internal/agentsessions`
- `scenarios/swarm-manager/cli`
- Session detail UI tests only if UI code changes.

Run:

```bash
cd scenarios/swarm-manager/api && go test ./internal/agentsessions -timeout 300s
cd scenarios/swarm-manager/cli && go test ./... -timeout 300s
```

If UI code changes:

```bash
cd scenarios/swarm-manager/ui && pnpm test -- SessionDetailsPage.test.tsx agent-session-service.test.ts agent-session-store.test.ts
```

### Scenario Lifecycle Tests

Use lifecycle commands only:

```bash
cd scenarios/workspace-sandbox && make test
cd scenarios/agent-manager && make test
cd scenarios/swarm-manager && make test
```

If these are too slow for every iteration, run targeted Go tests first and reserve lifecycle tests for pre-merge validation.

### End-to-End Validation

Start scenarios through lifecycle tooling:

```bash
vrooli scenario start workspace-sandbox
vrooli scenario start agent-manager
vrooli scenario start swarm-manager
```

Then run a controlled sandboxed agent-manager session that asks the agent to execute:

```bash
command -v vrooli
command -v swarm-manager
command -v buf
command -v protoc
command -v protoc-gen-go
command -v protoc-gen-connect-go
command -v protoc-gen-es
swarm-manager sessions --help
```

Validate:

- Agent-manager events include one final assistant message.
- The sandboxed process does not need manual `PATH` exports.
- Host requirement checks pass.
- Swarm-manager session details show the same single assistant response.
- `swarm-manager sessions get --id <session-id> --json` matches the UI-visible message count.

## Rollout And Validation

1. Land failing tests.
2. Implement transcript decoder state.
3. Implement workspace-sandbox env composition.
4. Update agent-manager launcher tests and contracts.
5. Strengthen swarm-manager session idempotence tests.
6. Run targeted tests.
7. Run scenario lifecycle tests.
8. Run one greenfield end-to-end sandboxed session.
9. Re-check the original run only as historical evidence; do not mutate or reinterpret its stored event history.

## Risks

- Changing the codec interface touches every runner codec. Mitigate by updating all codecs in one commit and relying on compile-time failures plus golden tests.
- Making `PATH` useful can accidentally make it too broad. Mitigate with an allowlisted, deterministic Vrooli-aware path instead of inheriting the developer shell.
- Some host tools may be symlinks into cache directories. Validate that the existing home overlay exposes symlink targets or fail with an explicit host requirement message that names the inaccessible target.
- Full end-to-end agent validation can be slow or provider-dependent. Keep deterministic unit and integration tests in CI, and use one live run as release validation.
- Stale local scenario binaries may keep confusing manual debugging. Treat them as unsupported for sandboxed agents unless lifecycle tooling validates freshness.

## Non-Goals

- No legacy transcript replay compatibility mode.
- No content-based event dedupe in agent-manager persistence as a substitute for correct parsing.
- No swarm-manager UI workaround for duplicated agent-manager events.
- No prompt-level workaround that tells agents to export paths manually.
- No direct execution of scenario APIs or binaries outside lifecycle tooling.
- No package installation as part of the implementation.

## Definition Of Done

- Agent-manager Claude transcript tests prove final assistant content is emitted once when assistant content and result lines both exist.
- Agent-manager still preserves result-only assistant text.
- Live tail plus final drain cannot duplicate assistant content.
- Workspace-sandbox Vrooli-aware env tests prove required CLIs and codegen tools are discoverable through deterministic `PATH`.
- Agent-manager sandbox launcher tests prove run identity and sandbox profile selection remain intact.
- Swarm-manager session refresh remains idempotent and session details show one assistant message.
- Targeted Go tests pass for agent-manager, workspace-sandbox, and swarm-manager session code.
- Scenario lifecycle tests pass for the three affected scenarios or any skipped lifecycle test has a documented blocker.
- A fresh sandboxed run validates command availability and produces non-duplicated agent-manager events.
