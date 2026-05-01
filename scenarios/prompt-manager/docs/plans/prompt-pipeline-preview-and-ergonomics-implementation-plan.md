# Prompt Pipeline Preview and Ergonomics Implementation Plan

## Purpose

Make prompt-manager's heartbeat prompt pipeline easier to inspect, harder to drift, and easier for agents to execute.

The backend already has the right architectural center: heartbeat execution, flat prompt preview, structured prompt preview, and team prompt matrix all flow through the same prompt builder. The remaining work is to make every consumer use that source of truth correctly, clarify the separate purpose of `member-context`, expose full prompt preview through the CLI, and reduce prompt cognitive load without removing capability.

## Required Reading

Future implementers should run:

```bash
prompt-manager skill read plan-skill-discovery implementation-plan-authoring documentation-health cli-steer api-steer seam-discovery-and-enforcement test react-coherence ux
```

Discovery command used while authoring this plan:

```bash
prompt-manager discover "prompt preview structured sections" "team member context CLI" "prompt ergonomics storage map" "UI API drift prevention" --complexity moderate
```

Discovery surfaced `documentation-health`, `seam-discovery-and-enforcement`, `test`, `interoperability-steer`, `api-steer`, `cli-steer`, `react-coherence`, `ux`, `cross-platform-readiness`, and `storage-steer`. The execution-relevant subset for this plan is the required-reading command above.

## Hard Rule

This is greenfield cleanup for the affected prompt-preview surfaces. Do not preserve stale UI labels, duplicate prompt-parsing logic, `Durable State` display concepts, or parallel old/new section ordering tables.

Keeping `member-context` is not a compatibility shim. It is a distinct standing-context API that intentionally omits the active heartbeat task. The cleanup is to name, document, and expose it honestly.

## Problem Statement

The prompt builder is mostly correct, but consumers are uneven:

- Heartbeat execution calls `Executor.BuildPrompt`, which calls `PromptBuilder.Build`.
- `POST /api/v1/prompt-preview` calls `Executor.BuildPrompt`, so it previews the same full prompt used by heartbeat execution.
- `POST /api/v1/prompt-preview-structured` calls `Executor.BuildPromptStructured`, which returns the same ordered prompt sections from the same builder.
- `GET /api/v1/teams/{id}/prompt-matrix` calls `Executor.BuildPromptStructured` for every member.
- `prompt-manager team member-context` calls the separate `BuildContext` path, which intentionally omits `HEARTBEAT.md`.

The drift is in client surfaces:

- `MemberPromptPipelineSection.tsx` calls flat `/prompt-preview`, then manually reparses markdown into hardcoded buckets.
- That UI parser still contains stale `Durable State` grouping and does not model `Storage Map`.
- `PromptSectionSchema` still has `team-durable-state` but not `team-operating-contract` or `team-storage-map`.
- `TeamPromptMatrixTab.tsx` still has a hardcoded section order/label table with `team-durable-state`.
- The CLI has `member-context` but no full prompt preview command, so operators and agents naturally reach for the wrong command when auditing actual runtime prompts.
- Many agent `AGENTS.md` files still instruct heartbeat-spawned agents to run `member-context`, even though the full runtime prompt already contains that context.
- For large teams, `Your Team Storage` duplicates long plan-of-record path lists already present in the resolved operating contract, increasing cognitive load without adding much decision value.

## Scope

In scope:

- Make UI prompt pipeline and prompt matrix render backend-provided structured sections directly.
- Update UI schemas and section-card metadata so current and future section kinds render without breaking.
- Add CLI commands for full prompt preview, structured prompt preview, and team prompt matrix.
- Clarify `member-context` semantics in CLI help, API docs, heartbeat docs, and agent bootstrap prose.
- Improve prompt ergonomics with a generated execution brief and a less duplicative storage rendering.
- Add tests that protect source-of-truth ordering and prevent stale UI/CLI labels from returning.

Out of scope:

- Changing heartbeat execution scheduling.
- Changing team storage ontology semantics.
- Removing `member-context`.
- Adding API/CLI enforcement of write permissions.
- Redesigning the whole team editor UI.

## Current Technical Context

Primary backend files:

- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder.go] - `Build`, `BuildStructured`, `BuildContext`, section ordering, storage-map rendering.
- [CODE: scenarios/prompt-manager/api/heartbeat/executor.go] - `BuildPrompt`, `BuildPromptStructured`, `BuildContext`.
- [CODE: scenarios/prompt-manager/api/heartbeat/handlers.go] - `/prompt-preview`, `/prompt-preview-structured`, `/teams/{id}/prompt-matrix`, and member context handler.
- [CODE: scenarios/prompt-manager/api/main.go] - route registration.
- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder_test.go] - builder equivalence and section ordering tests.
- [CODE: scenarios/prompt-manager/api/heartbeat/handlers_preview_test.go] - flat preview endpoint tests.
- [CODE: scenarios/prompt-manager/api/heartbeat/handlers_preview_structured_test.go] - structured preview endpoint tests.
- [CODE: scenarios/prompt-manager/api/heartbeat/handlers_prompt_matrix_test.go] - matrix endpoint tests.

Primary CLI files:

- [CODE: scenarios/prompt-manager/cli/teams/teams.go] - team subcommands, `member-context`, command help.
- [CODE: scenarios/prompt-manager/cli/PARITY_AUDIT.md] - current API/CLI coverage notes.
- [CODE: scenarios/prompt-manager/docs/reference/heartbeat-cli.md] - heartbeat CLI docs.

Primary UI files:

- [CODE: scenarios/prompt-manager/ui/src/components/editor/MemberPromptPipelineSection.tsx] - drift-prone flat prompt parser and pipeline cards.
- [CODE: scenarios/prompt-manager/ui/src/components/editor/MemberPromptPreview.tsx] - already uses structured preview.
- [CODE: scenarios/prompt-manager/ui/src/components/editor/teamTabs/TeamPromptMatrixTab.tsx] - matrix UI with stale section kind order/labels.
- [CODE: scenarios/prompt-manager/ui/src/components/editor/tabs/SectionCard.tsx] - reusable section display metadata with stale `team-durable-state`.
- [CODE: scenarios/prompt-manager/ui/src/lib/schemas/agent.schema.ts] - prompt section Zod enum is stale.
- [CODE: scenarios/prompt-manager/ui/src/lib/api.ts] - API client already exposes flat preview, structured preview, and matrix calls.
- [CODE: scenarios/prompt-manager/ui/src/services/agentService.ts] - service wrappers.

Docs and prompt prose:

- [CODE: scenarios/prompt-manager/docs/concepts/HEARTBEATS.md] - prompt pipeline docs.
- [CODE: scenarios/prompt-manager/docs/reference/api-endpoints.md] - endpoint docs.
- `scenarios/prompt-manager/store/agents/*/AGENTS.md` - many files instruct heartbeat-spawned agents to run `member-context`.

Observed current facts:

- `POST /api/v1/prompt-preview` is the accurate full runtime prompt preview.
- `POST /api/v1/prompt-preview-structured` is the accurate ordered section view.
- `prompt-manager team member-context <team-id> <agent-id>` is taskless standing context, not a full runtime prompt.
- UI schemas and display labels lag behind the backend section vocabulary.

## Target End State

1. Backend remains the source of truth for prompt section order.
2. UI prompt pipeline renders the exact ordered `sections[]` returned by `/prompt-preview-structured`.
3. UI does not hardcode section presence or order. It may map known `section.kind` values to icons/labels, but unknown kinds still render using `section.label`.
4. UI schemas accept backend section kinds without breaking when new section kinds are added.
5. CLI exposes:
   - full prompt preview for one member
   - structured prompt preview for one member
   - team prompt matrix for all members
6. `member-context` is documented as "standing context without active heartbeat task."
7. Heartbeat-spawned agents are no longer routinely told to run `member-context` redundantly.
8. Full prompts include a concise generated orientation before long policy/storage sections.
9. `Your Team Storage` remains useful but no longer duplicates large plan-of-record lists already present in the resolved operating contract.

## Contract Decisions

### Prompt Surfaces

Use these names consistently:

| Surface | Includes `HEARTBEAT.md`? | Shape | Intended use |
|---|---:|---|---|
| Full prompt preview | yes | flat markdown or structured sections | Audit/debug exactly what a heartbeat run receives |
| Structured prompt preview | yes | ordered `PromptSection[]` | UI rendering, tests, prompt pipeline cards |
| Team prompt matrix | yes | ordered `PromptSection[]` per member | Cross-member audit and drift detection |
| Member context | no | flat markdown standing context | External/leader/taskless context reuse |

### Section Ordering

The backend owns order. UI must render the `sections[]` order as returned.

Do not maintain a UI order table that tries to mirror backend ordering. A UI label/icon map is allowed, but ordering must come from the backend array.

### UI Schema Strictness

Change `PromptSectionSchema.kind` from a closed enum to a string with known-kind metadata in UI code.

Rationale: the API is already the typed source of truth in Go. A closed frontend enum creates an unnecessary second schema authority and causes runtime validation failures when the backend adds a legitimate section kind.

The UI may still define a TypeScript union for known kinds if useful, but parsing should accept unknown strings and render them with fallback metadata.

### CLI Commands

Add commands under `prompt-manager team`:

```bash
prompt-manager team prompt-preview <team-id> <agent-id> [--json]
prompt-manager team prompt-preview-structured <team-id> <agent-id> [--json]
prompt-manager team prompt-matrix <team-id> [--json]
```

Default output should be human-friendly:

- `prompt-preview`: print the flat prompt directly by default.
- `prompt-preview-structured`: print ordered section summaries plus content, unless `--json` is passed.
- `prompt-matrix`: print a table of members x section kinds with character counts by default; `--json` emits the API response.

Keep `member-context`, but update help text:

```text
member-context <team-id> <agent-id>  Get standing member context without HEARTBEAT.md
```

### Prompt Ergonomics

Add a generated `# Execution Brief` section before long contract/storage sections.

Suggested placement:

```text
Agent Files
Team Charter
Execution Brief
Resolved Operating Contract
Responsibilities
Org Context
Coordination
Storage Map
Inbox
Previous Handoff
Heartbeat Task
```

Suggested contents:

```markdown
# Execution Brief

Member: `<agent-id>`
Team: `<team-id>`
Lane: <member contract lane>

This heartbeat's concrete task is defined at the end of this prompt in `# Heartbeat Task (HEARTBEAT.md)`.

Use the sections below as operating context. If a section conflicts with the heartbeat task, follow the authority order in `# Storage Map`.
```

If the builder can cheaply derive a task title from the heartbeat file's first heading, include:

```markdown
Task: <first heading after "# Heartbeat:">
```

Do not summarize or rewrite the full heartbeat task in the execution brief. The goal is orientation, not a second task source.

### Storage Map Compression

Keep the universal Continue / Observe / Propose / Operate wording unchanged.

Change `## Your Team Storage` to reduce duplication:

- Continue to list notebooks exactly, because notebook routing is semantically important and usually small.
- Continue to list team working state exactly, because owners and update modes matter.
- For plan-of-record docs:
  - If the list is small, exact rendering is fine.
  - If the list is large, render a compact grouped summary and point to `## Document Authority` in the resolved operating contract for exact paths.

Example:

```markdown
Plan of record, read/propose only:
- 23 docs under `docs/monetization/`
  Policy: `operator-curated-via-decisions`
  Exact paths: see `## Document Authority` above.
```

Threshold recommendation:

- exact list for `<= 8` plan-of-record docs
- grouped summary for `> 8` plan-of-record docs

Grouping recommendation:

- group by first stable directory prefix, e.g. `docs/monetization/`, `docs/marketing/`, `docs/narrative/`
- keep individually listed root files such as `VISION.md`
- do not group files with mixed write policies unless the group output makes the policy split explicit

This preserves capability because exact paths remain in the resolved operating contract, while the storage map keeps the mental model readable.

## Implementation Strategy

### Phase 1 - Backend Contract Verification and Minor Extension

Owner profile: backend/API worker.

Files:

- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder.go]
- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder_test.go]
- [CODE: scenarios/prompt-manager/api/heartbeat/handlers_preview_structured_test.go]
- [CODE: scenarios/prompt-manager/api/heartbeat/handlers_prompt_matrix_test.go]

Tasks:

1. Confirm `Build`, `BuildStructured`, and `BuildContext` are intentionally distinct and documented in comments.
2. Add `Execution Brief` generation in `buildSectionList` after team charter and before resolved operating contract.
3. Add a new `PromptSection.Kind` value: `execution-brief`.
4. Ensure `Build()` and `BuildStructured()` remain equivalent when reassembled.
5. Keep `BuildContext()` omitting `heartbeat-task`, but including `execution-brief` only if it does not claim to include the active task body. If that wording feels misleading in taskless context, render a context-specific sentence:

   ```markdown
   The active heartbeat task is intentionally omitted from member context.
   ```

6. Add/adjust tests for:
   - full preview includes `execution-brief`
   - structured preview includes `execution-brief` in the correct order
   - context preview omits heartbeat task
   - full preview includes heartbeat task
   - matrix entries include `execution-brief`

Acceptance:

- Backend tests prove the prompt builder remains the sole source of section order.
- No UI or CLI needs to infer ordering from flat markdown.

### Phase 2 - Storage Map Ergonomics

Owner profile: backend/contract worker.

Files:

- [CODE: scenarios/prompt-manager/api/teamcontract/contract.go]
- [CODE: scenarios/prompt-manager/api/teamcontract/contract_test.go]
- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder.go]
- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder_test.go]

Tasks:

1. Locate the renderer for `## Your Team Storage`.
2. Add grouped plan-of-record rendering for large plan-of-record lists.
3. Preserve exact notebook and team working-state rendering.
4. Ensure the grouped plan-of-record summary references the exact-path source:

   ```text
   Exact paths: see `## Document Authority` above.
   ```

5. Add tests for:
   - small plan-of-record lists render exact paths
   - large same-policy lists group by stable prefix
   - mixed-policy lists do not collapse into misleading groups
   - notebooks still render exact curator/promotion context
   - team working state still renders exact owner/kind/use text

Acceptance:

- Marketing and monetization storage map sections shrink materially.
- Agents still receive exact plan-of-record paths in the resolved operating contract.
- Storage semantics do not change.

### Phase 3 - CLI Prompt Preview Surfaces

Owner profile: CLI worker.

Files:

- [CODE: scenarios/prompt-manager/cli/teams/teams.go]
- CLI tests in the same package, if present or added.
- [CODE: scenarios/prompt-manager/docs/reference/heartbeat-cli.md]
- [CODE: scenarios/prompt-manager/cli/PARITY_AUDIT.md]

Tasks:

1. Add response types for:
   - `PromptPreviewResponse`
   - `StructuredPromptPreviewResponse`
   - `TeamPromptMatrixResponse`
   Reuse existing shapes where already defined.
2. Add subcommands:

   ```text
   prompt-preview
   prompt-preview-structured
   prompt-matrix
   ```

3. Wire them to:

   ```text
   POST /prompt-preview
   POST /prompt-preview-structured
   GET /teams/{id}/prompt-matrix
   ```

4. Keep default output human-readable.
5. Add `--json` to each command.
6. Update `member-context` help to say it omits `HEARTBEAT.md`.
7. Update docs and parity audit to mark the API endpoints as CLI-covered.
8. Add tests around command routing and output formatting if the CLI package has existing command tests; otherwise add focused tests using the existing `appctx.Context` seam.

Acceptance:

```bash
prompt-manager team prompt-preview director-swarm vision-walk-prep
prompt-manager team prompt-preview-structured director-swarm vision-walk-prep
prompt-manager team prompt-matrix director-swarm
prompt-manager team member-context director-swarm vision-walk-prep
```

The first command includes `# Heartbeat Task`; the fourth does not.

### Phase 4 - UI Structured Pipeline Cutover

Owner profile: UI/frontend worker.

Files:

- [CODE: scenarios/prompt-manager/ui/src/components/editor/MemberPromptPipelineSection.tsx]
- [CODE: scenarios/prompt-manager/ui/src/components/editor/MemberPromptPreview.tsx]
- [CODE: scenarios/prompt-manager/ui/src/components/editor/teamTabs/TeamPromptMatrixTab.tsx]
- [CODE: scenarios/prompt-manager/ui/src/components/editor/tabs/SectionCard.tsx]
- [CODE: scenarios/prompt-manager/ui/src/lib/schemas/agent.schema.ts]
- [CODE: scenarios/prompt-manager/ui/src/lib/api.ts]
- [CODE: scenarios/prompt-manager/ui/src/services/agentService.ts]

Tasks:

1. Change `MemberPromptPipelineSection` to call `previewAgentPromptStructured`.
2. Delete `parsePromptSections`, `stripHeader`, and the hardcoded `PIPELINE_SECTIONS` presence/order list.
3. Render `response.sections` in returned order.
4. Keep the full flat prompt copy affordance by reassembling structured sections with `\n\n---\n\n`, or call flat preview only for the copy action if necessary. Prefer reassembly to avoid a second API call.
5. Use `section.label` as the primary display title.
6. Use kind metadata only for icon/color/category.
7. Update `PromptSectionSchema.kind` to accept `z.string()`.
8. Update known kind metadata:

   ```text
   agent-file
   team-shared-charter
   execution-brief
   team-operating-contract
   team-responsibilities
   team-org-context
   team-coordination
   team-storage-map
   team-inbox
   last-handoff
   heartbeat-task
   ```

9. Remove `team-durable-state` labels from active UI displays.
10. Update `TeamPromptMatrixTab` to derive active kinds in first-seen backend order, not from a hardcoded order list.
11. Matrix labels should use `section.label` or backend-derived labels, not stale kind labels.
12. Ensure unknown future section kinds render instead of failing validation.

Acceptance:

- UI pipeline order changes automatically if backend section order changes.
- UI no longer contains active `Durable State` display text.
- Structured prompt preview, pipeline, and matrix agree on section order and labels.

### Phase 5 - Member Context Semantics and Prompt Prose Cleanup

Owner profile: prompt/docs worker.

Files:

- `scenarios/prompt-manager/store/agents/*/AGENTS.md`
- [CODE: scenarios/prompt-manager/docs/concepts/HEARTBEATS.md]
- [CODE: scenarios/prompt-manager/docs/reference/api-endpoints.md]
- [CODE: scenarios/prompt-manager/docs/reference/heartbeat-cli.md]
- possibly [CODE: scenarios/prompt-manager/api/interop/claude_code.go]

Tasks:

1. Update docs to define:
   - full prompt preview
   - structured prompt preview
   - prompt matrix
   - member context
2. Replace language that implies `member-context` is the full heartbeat prompt.
3. Audit `store/agents/*/AGENTS.md` for instructions like:

   ```text
   Run `prompt-manager team member-context ...`
   ```

4. For heartbeat-spawned members, replace with:

   ```markdown
   The full heartbeat prompt already includes your member context. Use `prompt-manager team member-context <team-id> <agent-id>` only when an external workflow needs standing context without the active heartbeat task.
   ```

   Or remove the line if the remaining start-of-session instructions are sufficient.

5. Keep member-context instructions only where the file is clearly intended for an external/manual agent that is not spawned through the heartbeat pipeline.
6. Update `api/interop/claude_code.go` wording if it instructs external agents to use `member-context`; make clear it is taskless context.

Acceptance:

- Agents are not routinely told to fetch context they already have.
- Operators and future agents can explain the difference between full prompt preview and member context.

### Phase 6 - Documentation and Manifest

Owner profile: documentation worker.

Files:

- [CODE: scenarios/prompt-manager/docs/concepts/HEARTBEATS.md]
- [CODE: scenarios/prompt-manager/docs/reference/api-endpoints.md]
- [CODE: scenarios/prompt-manager/docs/reference/heartbeat-cli.md]
- [CODE: scenarios/prompt-manager/docs/manifest.json]
- This plan file.

Tasks:

1. Update `HEARTBEATS.md` prompt pipeline section to say the UI uses structured preview sections and renders backend order.
2. Add API docs for `/prompt-preview-structured` and `/teams/{id}/prompt-matrix` if incomplete.
3. Update `/prompt-preview` docs to say it matches heartbeat execution.
4. Update member-context docs to say it omits `HEARTBEAT.md`.
5. Register this plan in `docs/manifest.json`.

Acceptance:

- Docs no longer state "operators use member-context" as a substitute for full prompt preview.
- Docs describe backend as the source of truth for prompt section order.

### Phase 7 - Validation and Cleanup

Owner profile: validation worker.

Commands:

```bash
cd scenarios/prompt-manager/api && go test ./...
cd scenarios/prompt-manager/cli && go test ./...
cd scenarios/prompt-manager/ui && pnpm test
cd scenarios/prompt-manager && make test
```

If UI test time is high, run targeted tests first, then full scenario test through `make test` before final acceptance.

Search checks:

```bash
rg "team-durable-state|Durable State" scenarios/prompt-manager/ui scenarios/prompt-manager/cli scenarios/prompt-manager/docs
rg "member-context .*full|full member context|operators use `team member-context`" scenarios/prompt-manager
rg "prompt-preview|prompt-preview-structured|prompt-matrix" scenarios/prompt-manager/docs/reference scenarios/prompt-manager/cli/PARITY_AUDIT.md
```

Manual API/CLI checks:

```bash
prompt-manager team prompt-preview director-swarm vision-walk-prep | rg "# Heartbeat Task"
prompt-manager team member-context director-swarm vision-walk-prep | rg "# Heartbeat Task" && false || true
prompt-manager team prompt-preview-structured director-swarm vision-walk-prep --json
prompt-manager team prompt-matrix director-swarm
```

Manual UI checks:

- Open a member's prompt pipeline.
- Verify sections render in backend order.
- Verify `Storage Map` appears as its own section.
- Verify no `Durable State` label appears.
- Verify full prompt copy contains the same section order as the structured view.
- Open team prompt matrix.
- Verify active columns reflect backend section kinds, including `execution-brief` and `team-storage-map`.

Scenario lifecycle:

```bash
cd scenarios/prompt-manager && make stop && make start && make test
```

Use scenario Makefiles rather than direct binary execution.

## Testing Plan

### Backend Tests

Add or update tests for:

- `Build()` includes `execution-brief` and `heartbeat-task`.
- `BuildContext()` includes standing context but not `heartbeat-task`.
- `BuildStructured()` section order matches `Build()` reassembly.
- `/prompt-preview` includes heartbeat task.
- `/prompt-preview-structured` returns ordered current section kinds.
- `/teams/{id}/prompt-matrix` includes current section kinds and handles member-specific errors.
- Storage map compression preserves exact notebook and team working-state rendering.
- Large plan-of-record lists group only when safe.

### CLI Tests

Add tests for:

- subcommand registration
- human-readable default output
- `--json` output
- `prompt-preview` includes `# Heartbeat Task`
- `member-context` does not include `# Heartbeat Task`
- help text distinguishes full preview from standing context

### UI Tests

Add or update tests for:

- `PromptSectionSchema` accepts unknown future section kinds.
- `MemberPromptPipelineSection` renders sections in returned order.
- no flat markdown parsing is needed for section cards.
- matrix derives active section kinds from response order.
- `SectionCard` renders `team-storage-map`, `team-operating-contract`, and `execution-brief` metadata.
- unknown kinds render with fallback metadata and `section.label`.

### Documentation Checks

Run:

```bash
jq empty scenarios/prompt-manager/docs/manifest.json
knowledge-observatory docs audit scenarios/prompt-manager
```

If docs audit reports unrelated historical issues, document them in the final implementation summary rather than hiding them.

## Rollout Checklist

- [ ] `Execution Brief` appears in full and structured previews.
- [ ] Full prompt preview remains identical to heartbeat execution prompt.
- [ ] Structured preview remains ordered by backend section builder.
- [ ] `member-context` remains taskless and is documented as taskless.
- [ ] CLI has full preview, structured preview, and matrix commands.
- [ ] UI pipeline renders structured sections directly.
- [ ] UI prompt matrix derives order from backend responses.
- [ ] UI active labels no longer mention `Durable State`.
- [ ] Storage map is shorter for marketing/monetization without losing exact paths from the resolved contract.
- [ ] Agent bootstrap prose no longer tells heartbeat-spawned agents to redundantly fetch member context.
- [ ] API, CLI, UI, and docs agree on terminology.
- [ ] Tests and scenario validation pass.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| UI silently drops future backend section kinds | Parse `kind` as string, render unknown kinds with fallback metadata and backend `label`. |
| Another UI order table drifts | Render `sections[]` in returned order. Use no client order table for the primary pipeline. |
| `member-context` remains misunderstood | Rename help/docs around "standing context without HEARTBEAT.md"; add full prompt preview CLI command. |
| Storage map compression hides needed paths | Keep exact plan-of-record paths in `Resolved Operating Contract`; only compress duplicate plan-of-record rendering in `Your Team Storage`. |
| Execution brief becomes a second source of task truth | Make it orienting only; point to `# Heartbeat Task` for the concrete task. |
| Prompt gets longer due execution brief | Offset by storage-map compression and AGENTS.md cleanup. |
| UI validates too loosely | Keep schema loose for section kind only; retain typed structure for label/sourcePath/content. |

## Non-Goals and Prohibited Patterns

Non-goals:

- No scheduling or heartbeat runtime behavior changes.
- No new storage ontology.
- No write-permission enforcement.
- No broad team-editor redesign.

Prohibited patterns:

- No active UI display labels for `Durable State`.
- No UI code that parses flat markdown to infer section order when structured sections are available.
- No closed frontend enum that can reject valid backend section kinds.
- No CLI command that describes member-context as the full runtime prompt.
- No duplicate execution-task summary that can conflict with `HEARTBEAT.md`.

## Suggested Multi-Agent Work Split

1. Backend prompt worker:
   - Owns execution brief and storage-map compression in API code/tests.
2. CLI worker:
   - Owns team prompt preview commands and CLI docs.
3. UI schema/rendering worker:
   - Owns schema updates, `SectionCard`, prompt pipeline, and matrix.
4. Prompt prose/docs worker:
   - Owns `AGENTS.md` cleanup and docs terminology.
5. Validation worker:
   - Owns full test pass, grep audits, scenario Makefile validation, and final missed-drift cleanup.

Avoid parallel edits to the same files. The repo may already contain unrelated dirty changes; do not revert them.

## Definition of Done

This work is complete when:

- Backend prompt builder remains the single source of truth for full prompt content and section order.
- UI pipeline and matrix render backend structured sections directly.
- CLI exposes the same full prompt preview surfaces already available through the API.
- `member-context` is clearly differentiated as taskless standing context.
- Prompt ergonomics improve through a concise execution brief and reduced storage duplication.
- No active UI/CLI/docs surface treats `Durable State` as a current section.
- Tests cover backend, CLI, UI, and documentation behavior.
- `cd scenarios/prompt-manager && make test` passes after restarting the scenario through the lifecycle system.
