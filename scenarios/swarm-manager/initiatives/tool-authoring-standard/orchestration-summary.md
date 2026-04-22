# Tool Authoring Standard — Meta-Orchestrator Summary

## Source

Brainstorming session (2026-04-22) on making agent-inbox's conversational surface easy for non-technical users. The user observed that tools are currently only implemented in two scenarios (`agent-manager`, `scenario-to-cloud`), cost ~100 lines of hand-written protobuf per tool, have no canonical file location, no auditor rule, no authoring skill, and no template scaffolding. Result: adoption is low and unverifiable. This initiative fixes the authoring side so future scenarios can expose tools cheaply and fleet adoption becomes scannable.

Sibling initiatives spawned from the same session: `cli-conversational-surface`, `widget-standard`, `agent-inbox-unified-retrieval`. All four are independently authorable but converge at the retrieval layer.

## Shared Decisions (apply across all four sibling initiatives)

1. **Proto-first.** Contracts live in `packages/proto/schemas/agent-inbox/v1/domain/*.proto`. Tool proto exists today; widget proto will be added by the widget initiative.
2. **Manifest-free.** No `tools: []` array in `service.json`. Runtime endpoints + source-code scanning are authoritative; the manifest stays minimal.
3. **Fewer packages.** Extend existing shared packages (`api-core`, `cli-core`, whatever the shared UI package becomes) rather than carving new ones; only create a new package if research proves it's necessary.
4. **Auditor comparison, not manifest declaration.** Scenario-auditor rules compare code (e.g., `api/tools/` directory) against runtime (`/api/v1/tools` endpoint). Drift fails.
5. **Static embedding extraction** is the default for retrieval indexes wherever possible. Runtime-only extraction is reserved for cases where static can't capture the signal.

## Scope of This Initiative

Authoring side only. Discovery, indexing, and retrieval live in `agent-inbox-unified-retrieval`.

### What we're building

- **Shared Go toolkit** inside an existing `packages/` home (likely `packages/api-core` or generated alongside the existing proto-generated Go). Builder helpers collapse ~100 lines of hand-written `ToolDefinition` initialization into ~10 lines.
- **Canonical scenario location**: `api/tools/` directory. One file per tool (or grouped by category — decide in research). Each file declares the tool, its parameters schema, and its handler.
- **Scenario-auditor rule**: if `api/tools/` exists, the scenario must expose `/api/v1/tools` at runtime and the returned manifest must match the declared tools in source. Mismatch = fail.
- **Prompt-manager skills**:
  - `tool-authoring` — step-by-step guide for adding a tool to a scenario.
  - `tool-adoption-audit` — fleet scan producing a report of which scenarios expose tools, how many, and any drift.
- **Template refresh**: `react-vite + go-api` template ships with `api/tools/example.go` and a README section pointing agents at the authoring skill.

### What we're NOT building

- Any changes to agent-inbox consumption — that's `agent-inbox-unified-retrieval`.
- CLI-side discovery — that's `cli-conversational-surface`.
- Non-Go toolkit implementations. Go is the only API language in use today; future languages get sibling `packages/api-core-*` packages per the established pattern.
- Service.json manifest declarations.

## Anticipated Items (to be created next)

- `research/tool-authoring-toolkit-design` — decide toolkit shape (codegen vs. hand-written builders), canonical file layout, tool-grouping conventions, choice of package home, boilerplate target.
- `execute/tool-authoring-go-toolkit` — implement the shared Go toolkit.
- `execute/tool-authoring-auditor-rule` — scenario-auditor rule comparing `api/tools/` to `/api/v1/tools`.
- `execute/tool-authoring-prompt-manager-skills` — the two skills.
- `execute/tool-authoring-template-scaffolding` — update react-vite + go-api template.
- `execute/tool-authoring-migrate-existing` — convert `agent-manager` and `scenario-to-cloud` to the new toolkit as the first adopters (dogfood before fleet rollout).

## Cross-Initiative Dependencies

- **Consumed by** `agent-inbox-unified-retrieval` — provides the canonical tool descriptor shape the retrieval index embeds.
- **Parallel to** `cli-conversational-surface` and `widget-standard` — no shared code, but shared auditor-rule / skill / template conventions.
- **No upstream dependencies** — can start immediately.

## Open Questions Deferred to Workshop / Research

- **Codegen vs. builder helpers**: proto already defines types; is the best ergonomic story a generated Go file with nice constructors, or a hand-written builder package that wraps the proto types? Research item will prototype both on a representative tool and decide.
- **Package home**: `packages/api-core` vs. a subdirectory vs. a new `packages/tool-core`. Default is extend `api-core`; reconsider if research surfaces a clean boundary.
- **Tool-file grouping**: one tool per file (clearer) vs. grouped by category (fewer files). Authoring skill locks whichever wins.
- **Migration of existing tool authors**: `agent-manager` and `scenario-to-cloud` already hand-wrote their tools. Converting them is the dogfood step, not optional.
