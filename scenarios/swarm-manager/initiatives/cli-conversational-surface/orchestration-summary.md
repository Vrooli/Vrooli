# CLI as Conversational Surface — Meta-Orchestrator Summary

## Source

Brainstorming session (2026-04-22). The user pointed out that every scenario is already required to have a CLI built on the shared `cli-core` package, with standardized human output and token-efficient structured responses — this is exactly the infrastructure that lets coding agents use any scenario today. If that same surface could be exposed conversationally, non-technical users (and the future phone agent) could access every scenario from day one without waiting for each scenario to adopt proto tools.

Sibling initiatives: `tool-authoring-standard`, `widget-standard`, `agent-inbox-unified-retrieval`.

## Shared Decisions (apply across all four sibling initiatives)

1. **Proto-first.** Contracts live in `packages/proto/schemas/agent-inbox/v1/domain/*.proto`.
2. **Manifest-free.** No declarations in `service.json`; runtime and source-scanning are authoritative.
3. **Fewer packages.** Extend `cli-core` rather than creating a new one.
4. **Auditor comparison, not manifest declaration.**
5. **Static embedding extraction** where possible.

## Scope of This Initiative

A universal fallback conversational surface that works for every scenario automatically because every scenario already ships a cli-core CLI.

### What we're building

- **`introspect` subcommand in `cli-core`.** Emits a structured JSON command tree — commands, subcommands, flags, short/long descriptions, argument types, examples, output-shape hints. Every scenario CLI inherits this automatically.
- **Default human output contract preserved.** The existing behavior (`feedback_cli_default_human_output.md`: agent prompts use human output, not `--json`) is untouched. `introspect` is a *separate* surface for machine consumption, not a reshape of the default.
- **Synchronous-by-default execution model.** Agent-inbox invokes a CLI command by shelling out, captures the human-format output, streams back. No sync/async/approval metadata is assumed. When a scenario needs richer behavior, it `upgrades` that command by authoring a proper proto tool via `tool-authoring-standard` — both surfaces coexist, with proto tools taking precedence if both exist for the same name.
- **Scenario-auditor rule.** Every scenario CLI must respond to `introspect` with a valid command tree. cli-core enforcing this centrally is the cheapest version of this rule.

### What we're NOT building

- Any change to agent-inbox's retrieval logic — that's `agent-inbox-unified-retrieval`.
- A new transport or IPC mechanism. Invocation is plain shell-exec of the existing CLI binary.
- JSON output as default. The existing human-output contract stays.
- Per-command async/approval metadata. That's what proto tools are for — by design this surface is lossy-but-universal.

## Anticipated Items

- `research/cli-introspect-contract-design` — define the JSON schema for the command tree, how cli-core generates it automatically (reflection over cobra/whatever the CLI framework is), migration path for any bespoke CLI quirks.
- `execute/cli-core-introspect-command` — implement the subcommand in cli-core.
- `execute/cli-conversational-auditor-rule` — scenario-auditor rule validating every CLI responds to `introspect`.
- `execute/cli-conversational-agent-inbox-client` — the agent-inbox-side discovery + invocation client that reads `introspect` output and turns it into conversational descriptors consumable by the retrieval index. (This bleeds into agent-inbox-unified-retrieval's territory — either keep it here as the consumer glue, or move it there. Settle during research.)

## Cross-Initiative Dependencies

- **Consumed by** `agent-inbox-unified-retrieval` — CLI command descriptors become one of three surface types in the unified index.
- **Parallel to** `tool-authoring-standard` and `widget-standard`.
- **No upstream dependencies** — can start immediately.

## Design Intent: Long-Term Relationship Between CLI Surface and Proto Tools

Both exist on purpose:

- **Proto tools** are the *official* conversational contract. They carry async/approval/idempotency/examples — the metadata needed for rich agent reasoning. Good for high-value, high-frequency, or high-stakes operations.
- **CLI surface** is the *universal* conversational contract. It works for every scenario with zero per-scenario adoption cost once cli-core ships `introspect`. Good for the long tail — rarely-used but occasionally-critical commands.

Per the user's observation: it's plausible that in the long run the CLI surface alone is enough, because it gives conversational agents the same capabilities a coding agent already has. But until we prove that empirically, proto tools remain the stronger contract for anything where sync/async semantics or approval gating matters.

## Open Questions Deferred to Workshop / Research

- **CLI framework coverage**: does cli-core wrap a single CLI framework (cobra?), or does it need reflection hooks for multiple? Research surveys current scenario CLIs.
- **Output-shape hints**: can `introspect` describe *expected output patterns* (e.g., `emits a table of scenarios with columns X Y Z`) well enough to help the retrieval layer rank relevance? Or is that over-engineering?
- **Invocation sandboxing**: shell-execing a CLI from agent-inbox raises the usual `which CLI runs where` question (sandbox-aware paths — see `project_sandbox_aware_cli_tools.md`). Invocation path must respect those conventions.
- **Conflict resolution**: if a scenario exposes both a proto tool and a CLI command with the same name, proto wins. Workshop confirms.
