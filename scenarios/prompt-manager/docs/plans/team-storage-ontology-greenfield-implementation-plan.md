# Team Storage Ontology Greenfield Implementation Plan

## Purpose

Implement a greenfield, contract-derived storage ontology for prompt-manager teams so every heartbeat member receives a simple, uniform mental model for persistent state:

```text
Continue -> handoff
Observe  -> knowledge, notebook, friction evidence
Propose  -> decisions
Operate  -> team working state
```

The goal is to make persistent team memory more intuitive, more capable, and easier to maintain without reducing any team capability. This plan preserves the exact proposed agent-facing wording for the new prompt sections and defines the code, data, documentation, prompt-prose cleanup, and validation work needed to cut over cleanly.

## Required Reading

Future implementers must run:

```bash
prompt-manager skill read documentation-health implementation-plan-authoring team-shared-docs-design
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Recommended context:

```bash
prompt-manager skill read team-coordination-independent team-coordination-leader-led team-coordination-peer
prompt-manager skill read conversation-friction-analysis capability-extraction documentation-health
```

Primary code and docs:

- [CODE: scenarios/prompt-manager/api/teamcontract/contract.go] - operating contract schema, validation, member rendering
- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder.go] - heartbeat/member-context prompt assembly
- [CODE: scenarios/prompt-manager/api/teamcontract/contract_test.go] - bundled contract validation and prompt-prose drift checks
- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder_test.go] - prompt section ordering and content tests
- [CODE: scenarios/prompt-manager/store/schemas/team.schema.json] - team JSON schema
- [CODE: scenarios/prompt-manager/docs/concepts/MEMORY-PROMOTION.md] - existing notebook promotion ontology
- [CODE: scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md] - plan-of-record vs notebook design

## Hard Greenfield Rule

This is a hard cutover. Do not add compatibility shims, legacy aliases, fallback renderers, old/new dual paths, or dead transitional code.

Required behavior:

- Replace the old "Durable State" prompt section with the new "Storage Map" and "Your Team Storage" sections.
- Replace agent-facing "shared state" language with "team working state".
- Replace vague shared-state kind names with the final typed ontology.
- Update bundled teams, schemas, tests, docs, and prompt prose in one coherent cutover.
- Delete or rewrite obsolete tests and docs instead of preserving legacy expectations.

Prohibited patterns:

- No `legacyStorageMap`, `renderDurableStateV1`, `sharedStateKindAlias`, or equivalent compatibility layer.
- No support for both old and new shared-state kind vocabularies.
- No prompts that mention both "Durable State" and "Storage Map" as parallel concepts.
- No leftover docs instructing agents to reason from generic "shared state".

## Problem Statement

The current operating contract already centralizes decision contexts, knowledge topics, document authority, write rules, and member policy. However, the prompt still exposes persistence as a collection of partly overlapping surfaces:

- handoff
- knowledge log
- decisions
- notebooks
- plan-of-record docs
- generic shared state
- custom rolling artifacts
- custom append-only logs
- task boards
- operator inputs

Agents can follow these surfaces, but the mental model is too file-shaped. Teams also repeat generic contract/storage instructions in `AGENTS.md`, `TOOLS.md`, `RESPONSIBILITIES.md`, `HEARTBEAT.md`, and `shared/TEAM.md`. That creates prompt bloat and makes future storage improvements harder to explain consistently.

The desired model is behavior-shaped:

- **Continue**: preserve next-run continuity.
- **Observe**: record evidence, snapshots, lessons, and friction signals.
- **Propose**: queue reviewable changes.
- **Operate**: maintain live team-local work objects.

## Scope

In scope:

- Contract schema changes for typed team working state and storage-map rendering.
- Prompt builder changes to render the new universal and team-specific storage sections.
- Friction capture ontology and topic conventions.
- Updates to bundled team `operatingContract.documents.sharedState` entries.
- Prompt-prose cleanup across bundled agents and team member files.
- Documentation updates to concept/reference docs and manifest.
- Tests validating prompt wording, ordering, bundled data, schemas, and prompt prose cleanup.

Out of scope for this cutover:

- Enforcing write permissions at API/CLI boundaries.
- Changing persistence backends.
- Building new UI affordances beyond keeping existing UI/tests green if affected.
- Creating new teams or changing team missions.
- Altering decision approval semantics.

## Current Technical Context

### Prompt Assembly

`PromptBuilder.buildSectionList` currently assembles team prompts in this order:

```text
agent markdown
shared/TEAM.md
Resolved Operating Contract
RESPONSIBILITIES.md
org context
coordination guidance
Durable State
inbox
last handoff
HEARTBEAT.md
```

The current `buildDurableStateSection` renders generic CLI guidance for inbox, task board, decision log, knowledge log, and handoff.

### Contract Rendering

`teamcontract.RenderMember` currently renders:

- decision mode and pending ceiling
- member lane
- owned decision contexts
- decision caps
- required knowledge topics
- document authority:
  - plan-of-record docs
  - notebook docs
  - shared state
- allowed and forbidden writes
- safety rules
- task parameters

The document model already has:

```go
type Documents struct {
    PlanOfRecord []PlanOfRecordDocument `json:"planOfRecord"`
    Notebooks    []NotebookDocument     `json:"notebooks"`
    SharedState  []SharedStateDocument  `json:"sharedState"`
}
```

`SharedStateDocument.Kind` is currently a free-form string in practice. Bundled teams use values such as:

- `rolling-artifact`
- `append-only-log`
- `operator-state`
- `decision-stream`
- `knowledge-log`
- `handoff-history`
- `task-board`

### Existing Prompt Duplication

The bundled prompt prose repeatedly states generic storage/contract guidance, for example:

- "Apply the resolved operating contract..."
- "The resolved operating contract is authoritative for decision contexts, caps, source documents, shared state, write rules..."
- "Raise decisions only when allowed by the contract."
- "Write the required knowledge entry."
- "End with HANDOFF."
- "Use the resolved operating contract for source-document paths, writable surfaces..."

After this cutover, those generic statements should be generated by the prompt builder or contract renderer, not hand-authored into each member.

## Target End State

Every member prompt has:

1. A universal `# Storage Map` section with the exact wording in this plan.
2. A generated `## Your Team Storage` section derived from the team's operating contract.
3. A resolved operating contract that lists member-specific policy but does not force agents to infer storage ontology from raw path lists.
4. Member prose files that focus on lane, task loop, judgment, boundaries, and output shape, not generic storage rules.
5. Typed team working state, with clear semantics for each kind.
6. Friction signals captured as first-class observations, with meta-optimization able to mine `friction/*` knowledge topics.

## Exact Agent-Facing Section: Storage Map

The following wording must be preserved unless an implementation agent has a concrete reason to improve it and updates this plan or successor decision accordingly.

```markdown
# Storage Map

Use persistent storage only when information should survive this run.

## Continue

Use your final `## HANDOFF` for short-term continuity.

Write what your next run needs to know: what changed, what remains open, what to check first, and any blockers. Handoff is not canonical truth. It is next-run memory.

## Observe

Use the knowledge log for structured observations from this heartbeat: evidence, measurements, snapshots, findings, and concrete friction signals.

Use the notebook only for unresolved patterns, workarounds, or rough lessons that are not ready for durable structure. Notebook entries are debt, not authority. The curator later promotes or retires them.

If something expected was missing, broken, confusing, slow, undocumented, or harder than it should have been, capture it as friction. Mention one-off friction in handoff, write concrete friction to knowledge, append recurring workarounds to the notebook, and raise a decision only when the friction blocks work or points to a missing/broken capability.

## Propose

Use decisions for changes that need review.

Create a decision when something durable should change: plan-of-record docs, skills, actions, CLIs, team config, scenarios, backlog, or another member's operating surface. Include the proposed change, rationale, evidence, and target destination.

## Operate

Use team working state for live team objects you are assigned to maintain.

Working state is team-local operational memory: task boards, ledgers, registers, rolling audits, append-only logs, and operator input files. It is not automatically canonical outside the team. Update only the working-state files named in your operating contract.

## Authority Order

When sources disagree, prefer:

1. Operator instruction in the current run
2. Accepted plan-of-record docs
3. Accepted decisions
4. Team working state
5. Knowledge log evidence
6. Notebook entries
7. Handoff
```

## Exact Agent-Facing Section: Your Team Storage

The prompt builder should render this section immediately after the universal `# Storage Map` section, omitting empty categories.

Template:

```markdown
## Your Team Storage

Plan of record, read/propose only:
- `<path>`
  Policy: `<writePolicy>`
  Consumers: `<consumer-list-or-rationale>`

Notebook, append unresolved learning:
- `<path>`
  Curator: `<curatorMemberId>`
  Promotion context: `<promotionContext>`
  Posture: `<posture-or-debt>`

Team working state:
- `<path>`
  Kind: `<kind>`
  Owner: `<ownerMemberId-or-team>`
  Use for: `<kind-specific explanation>`

Always available:
- decisions: propose reviewable changes
- knowledge: record structured observations and friction signals
- handoff: preserve next-run continuity
```

Rendering rules:

- Use normalized repo-relative paths, matching current `NormalizePath` behavior.
- Omit `Consumers:` if both consumers and rationale are empty. Validation should already reject that for required plan-of-record docs.
- For plan-of-record docs with `writePolicy=read-only`, label as `read-only`.
- For notebooks, render curator and promotion context as mandatory.
- For team working state, render the kind-specific explanation from a centralized kind registry.
- Keep "Always available" even when the team has no custom documents.

Example output:

```markdown
## Your Team Storage

Plan of record, read/propose only:
- `docs/monetization/STRATEGY.md`
  Policy: `operator-curated-via-decisions`
  Consumers: `monetization`, `marketing-crew`

Notebook, append unresolved learning:
- `docs/marketing/notebook/README.md`
  Curator: `brand-manager`
  Promotion context: `notebook-promotion`
  Posture: `debt`

Team working state:
- `scenarios/prompt-manager/store/teams/monetization/shared/ledger.jsonl`
  Kind: `append-only-event-log`
  Owner: `financial-tracker`
  Use for: structured historical events or observations owned by the team
- `scenarios/prompt-manager/store/teams/monetization/shared/operator-inputs.json`
  Kind: `operator-input`
  Owner: `financial-tracker`
  Use for: operator-maintained inputs or state that agents may read and only assigned owners may maintain

Always available:
- decisions: propose reviewable changes
- knowledge: record structured observations and friction signals
- handoff: preserve next-run continuity
```

## Team Working State Kind Registry

Replace the current vague/shared-state kind vocabulary with this closed set:

| Kind | Meaning | Update mode | Typical owner | Agent-facing use text |
|---|---|---|---|---|
| `charter` | Human-readable team mission, principles, and team-specific framework | operator/team curated | team | team charter and durable team-specific principles |
| `task-board` | Live team task list | mutable | team or coordinator | live team tasks and coordination state |
| `decision-log` | Built-in governance queue | append/update via decision commands | team | reviewable proposed changes |
| `knowledge-log` | Built-in structured evidence memory | append/supersede by topic | team | structured observations, snapshots, and friction signals |
| `handoff-log` | Built-in continuity archive | automatic append | system | historical handoff archive |
| `working-register` | Mutable current list/table used in repeated team operations | append or update rows | named member/team | current operational list or register |
| `rolling-snapshot` | Curated current view updated over time | replace/update section or row | named member | current summarized view of recent evidence |
| `append-only-event-log` | Structured historical event stream | append-only | named member/team | structured historical events or observations owned by the team |
| `operator-input` | Operator-maintained or operator-approved mutable state | operator-maintained or assigned owner | operator/named member | operator-maintained inputs or state |

Cutover mapping:

| Old kind | New kind |
|---|---|
| `rolling-artifact` | choose `rolling-snapshot` or `working-register` per file |
| `append-only-log` | `append-only-event-log` |
| `operator-state` | `operator-input` |
| `decision-stream` | `decision-log` |
| `handoff-history` | `handoff-log` |
| `knowledge-log` | `knowledge-log` |
| `task-board` | `task-board` |

Recommended file-specific classifications:

- `shared/TEAM.md` -> `charter`
- `shared/decisions.jsonl` -> `decision-log`
- `shared/knowledge.jsonl` -> `knowledge-log`
- `shared/handoff-history.jsonl` -> `handoff-log`
- `shared/tasks.json` -> `task-board`
- Monetization `ledger.jsonl`, `market-scans.jsonl`, `opportunities.jsonl` -> `append-only-event-log` unless the owner mutates existing entries; if mutable, use `working-register`
- Monetization `operator-inputs.json` -> `operator-input`
- Marketing `audience-scans.jsonl`, `campaign-drafts.jsonl`, `publish-log.jsonl`, `published-improvements-log.jsonl`, `published-scenario-mentions.jsonl` -> `append-only-event-log`
- Marketing `coverage/*.json` -> `working-register` or `operator-input` depending on whether publisher or operator owns writes
- Meta-optimization `SKILL_AUDIT.md`, `TEAM_AUDIT.md`, `AGENT_AUDIT.md`, `TOOLCHAIN_SCAN.md`, `RUN_LESSONS.md` -> `rolling-snapshot`
- Meta-optimization `DEPRECATION_QUEUE.md`, `PROGRAMMATIC_CONVERSION_QUEUE.md` -> `working-register`
- Infra-health `RUNTIME_LESSONS.md`, `PLATFORM_AUDIT.md`, `AGING_SCAN.md` -> `rolling-snapshot`

## Friction Capture Contract

Add friction capture as an Observe subcase.

Friction definition:

```text
Something expected was missing, broken, confusing, slow, undocumented, misplaced, unstable, or harder than it should have been.
```

Storage routing:

```text
Agent hits friction
  -> minor / one-off:
       mention in HANDOFF only if useful next run

  -> concrete but not blocking:
       write knowledge topic `friction/<surface>/<YYYY-MM-DD>/<slug>`

  -> recurring pattern or workaround:
       append to notebook, if the team has one

  -> missing/broken capability that blocks work:
       raise capability-gap / bug / instrumentation decision

  -> platform/runtime/tooling issue:
       write friction knowledge; meta-optimization or infra-health can mine it
```

Required good friction note shape:

```markdown
Expected: ...
Actual: ...
Surface: ...
Workaround: ...
Impact: ...
Recurrence: one-off | recurring | unknown
```

Knowledge topic convention:

```text
friction/<surface>/<YYYY-MM-DD>/<slug>
```

Examples:

```text
friction/agent-manager/2026-05-01/stopped-during-dev-log
friction/prompt-manager-cli/2026-05-01/missing-member-context-field
friction/docs/2026-05-01/plan-of-record-path-unclear
friction/tooling/2026-05-01/validator-output-unstable
```

Meta-optimization ownership:

- `team-agent-optimizer` mines prompt, team, agent, storage-map, handoff, and operating-contract friction.
- `run-introspector` mines run failure, runtime execution, and prompt/task execution friction.
- `toolchain-validator` mines CLI, validator, test, and toolchain friction.
- `debt-curator` mines recurring friction workarounds in notebooks and shared artifacts for promotion/retirement.

## Contract Decisions

### Data Model

Keep the existing top-level `documents` shape:

```json
{
  "documents": {
    "planOfRecord": [],
    "notebooks": [],
    "sharedState": []
  }
}
```

Rationale: `sharedState` can remain the JSON field name as an internal contract category, but agent-facing prose must call it "team working state". This avoids needless schema churn while still removing the confusing phrase from prompts.

Change `SharedStateDocument.Kind` validation from effectively open text to a closed enum. No old values accepted.

Add kind metadata in code, preferably in `teamcontract`, for:

- validation
- rendering use text
- sorting/grouping if needed
- tests

Suggested Go shape:

```go
type TeamWorkingStateKind struct {
    ID          string
    Label       string
    UseText     string
    UpdateMode  string
}
```

### Prompt Sections

Replace `buildDurableStateSection` with a new renderer. Suggested names:

- `buildStorageMapSection(team *store.Team, agentID string) string`
- or a `teamcontract.RenderStorageMap(...)` function if centralizing all contract-derived prose is cleaner.

Preferred seam:

- Keep generic CLI command guidance in `heartbeat` because it depends on team runtime/capabilities.
- Put document classification rendering in `teamcontract` because it depends on contract types and normalized paths.

Avoid duplicating path rendering logic between `RenderMember` and prompt-builder-specific code. Reuse `NormalizePath`.

### Resolved Operating Contract

Keep `# Resolved Operating Contract`, but reduce document rendering ambiguity:

- It may keep policy details such as decision caps and required knowledge topics.
- It should not be the only place that teaches document semantics.
- It should call the custom document category "Team working state" rather than "Shared state".

### CLI/API

No endpoint or CLI behavior changes are required in this phase. Documentation may mention current commands, but do not add enforcement or new commands.

## Implementation Strategy

### Phase 1 - Contract Ontology and Rendering Core

Owner profile: backend/contract worker.

Files likely touched:

- [CODE: scenarios/prompt-manager/api/teamcontract/contract.go]
- [CODE: scenarios/prompt-manager/api/teamcontract/contract_test.go]
- [CODE: scenarios/prompt-manager/store/schemas/team.schema.json]

Tasks:

1. Add closed enum constants for team working state kinds.
2. Add kind metadata registry.
3. Update `validateDocuments` to reject unknown `sharedState.kind`.
4. Update `renderDocuments` to label `Team working state:` rather than `Shared state:`.
5. Add a renderer for team-specific storage map document lists if this is best centralized in `teamcontract`.
6. Add unit tests:
   - accepts every final kind
   - rejects old kinds
   - rejects unknown kinds
   - renders "Team working state" and never "Shared state"
   - renders notebook curator/promotion context
   - renders plan-of-record consumers/rationale

Acceptance:

- `go test ./teamcontract` passes.
- Old kind strings are rejected by tests.

### Phase 2 - Prompt Builder Storage Map Cutover

Owner profile: heartbeat/backend worker.

Files likely touched:

- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder.go]
- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder_test.go]

Tasks:

1. Delete/replace `buildDurableStateSection`.
2. Add universal `# Storage Map` section with exact wording from this plan.
3. Add generated `## Your Team Storage` subsection.
4. Preserve existing capability-sensitive CLI hints, but place them under a subordinate area only if still needed. Suggested heading:

   ```markdown
   ## Available Storage Commands
   ```

   This heading must be below the mental model, not above it.

5. Keep inbox/task/decision/knowledge/handoff command snippets if capabilities are enabled, but ensure they do not reintroduce "Durable State".
6. Update prompt ordering tests:

   ```text
   Team Charter
   Resolved Operating Contract
   Responsibilities
   Team Org Context
   Team Coordination
   Storage Map
   Team Inbox
   Previous Handoff
   Heartbeat Task
   ```

7. Add exact-content tests for:
   - `# Storage Map`
   - `## Continue`
   - `## Observe`
   - friction sentence
   - `## Propose`
   - `## Operate`
   - `## Authority Order`
   - `## Your Team Storage`
   - "Always available"
8. Add negative tests:
   - prompt must not contain `# Durable State`
   - prompt must not contain `Shared state:` in generated sections

Acceptance:

- `go test ./heartbeat` passes.
- Generated prompt contains exact storage wording.

### Phase 3 - Bundled Team Contract Cutover

Owner profile: data/schema worker.

Files likely touched:

- `scenarios/prompt-manager/store/teams/*/team.json`
- possibly `scenarios/prompt-manager/store/schemas/team.schema.json`

Tasks:

1. Replace all old shared-state kind values with final kinds.
2. Classify every custom file intentionally using the table in this plan.
3. Add missing shared-state entries for files that exist and should be surfaced in `Your Team Storage`.
4. Remove entries for stale files if any are no longer present.
5. Validate bundled teams using `TestBundledTeamContractsValidate`.

Important: because this is a hard greenfield cutover, do not keep old kind names in data or validation.

Acceptance:

- `rg '"kind": "rolling-artifact"|"append-only-log"|"operator-state"|"decision-stream"|"handoff-history"' scenarios/prompt-manager/store/teams` returns no matches.
- Bundled contract tests pass.

### Phase 4 - Prompt Prose Cleanup

Owner profile: prompt/docs worker.

Files likely touched:

- `scenarios/prompt-manager/store/agents/*/AGENTS.md`
- `scenarios/prompt-manager/store/agents/*/TOOLS.md`
- `scenarios/prompt-manager/store/teams/*/shared/TEAM.md`
- `scenarios/prompt-manager/store/teams/*/members/*/RESPONSIBILITIES.md`
- `scenarios/prompt-manager/store/teams/*/members/*/HEARTBEAT.md`
- [CODE: scenarios/prompt-manager/api/teamcontract/contract_test.go]

Cleanup rules:

Keep in `HEARTBEAT.md`:

- member-specific task loop
- domain-specific output shape
- domain-specific stop conditions
- artifact schemas when the member owns a custom file

Remove or shorten from `HEARTBEAT.md`:

- generic "raise decisions only when allowed by the contract"
- generic "write required knowledge"
- generic "end with HANDOFF"
- generic "apply resolved operating contract"

Keep in `RESPONSIBILITIES.md`:

- member lane
- judgment rules
- boundaries
- useful skills

Remove or shorten from `RESPONSIBILITIES.md`:

- repeated contract authority boilerplate
- repeated source docs/write rules/caps wording

Keep in `TEAM.md`:

- mission
- coordination pattern
- human-readable principles
- team-specific failure modes

Remove or shorten from `TEAM.md`:

- generic storage explanations now owned by generated Storage Map
- repeated plan-of-record/notebook rules unless team-specific
- generic "shared state" language

Keep in `AGENTS.md`:

- start-of-session bootstrap
- `prompt-manager team member-context ...`
- unusually important workflow specifics

Remove or shorten from `AGENTS.md`:

- duplicated task loop when `HEARTBEAT.md` owns it
- duplicated contract/storage boilerplate

Keep in `TOOLS.md`:

- actual tool access and primary skills
- domain-specific usage notes

Remove or shorten from `TOOLS.md`:

- generic source-document/writable-surface/capability-gap routing boilerplate

Test updates:

Expand `TestBundledPromptProseDoesNotRestateContractPolicy` to forbid:

- `shared state`
- `Shared State`
- `Durable State`
- `resolved operating contract is authoritative for decision contexts`
- `source documents, shared state, write rules`
- repeated generic handoff/knowledge/decision boilerplate where feasible

Use judgment: do not forbid valid domain phrases like "shared/TEAM.md" path references.

Acceptance:

- Bundled prompt prose is materially shorter and more domain-specific.
- Generic storage ontology appears once, generated by code.

### Phase 5 - Friction Capture Integration

Owner profile: meta-optimization/prompt worker.

Files likely touched:

- `scenarios/prompt-manager/store/teams/meta-optimization/team.json`
- `scenarios/prompt-manager/store/teams/meta-optimization/members/team-agent-optimizer/HEARTBEAT.md`
- `scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md`
- `scenarios/prompt-manager/store/teams/meta-optimization/members/toolchain-validator/HEARTBEAT.md`
- `scenarios/prompt-manager/store/teams/meta-optimization/members/debt-curator/HEARTBEAT.md`
- matching `RESPONSIBILITIES.md` files if needed

Tasks:

1. Add `friction/<surface>/<YYYY-MM-DD>/<slug>` to relevant `knowledgeTopics`.
2. Add task loop steps for mining friction:
   - `team-agent-optimizer`: prompt/team/agent/storage-map friction
   - `run-introspector`: run execution friction
   - `toolchain-validator`: CLI/tool/test/validator friction
   - `debt-curator`: recurring notebook/workaround friction
3. Keep these additions domain-specific. Do not restate the universal friction wording already generated in Storage Map.
4. Add examples only if needed for the member's lane.

Acceptance:

- Meta-optimization members have clear ownership for mining friction.
- All agents receive generic friction capture guidance via Storage Map.

### Phase 6 - Documentation Cutover

Owner profile: documentation worker.

Files likely touched:

- [CODE: scenarios/prompt-manager/docs/concepts/MEMORY-PROMOTION.md]
- [CODE: scenarios/prompt-manager/docs/concepts/TEAM-EXECUTION.md]
- [CODE: scenarios/prompt-manager/docs/reference/api-endpoints.md]
- [CODE: scenarios/prompt-manager/docs/reference/cli-commands.md]
- [CODE: scenarios/prompt-manager/docs/reference/heartbeat-api.md]
- [CODE: scenarios/prompt-manager/docs/reference/heartbeat-cli.md]
- [CODE: scenarios/prompt-manager/docs/README.md]
- [CODE: scenarios/prompt-manager/docs/manifest.json]
- [CODE: scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md]

Tasks:

1. Update docs to introduce the storage ontology:

   ```text
   Continue, Observe, Propose, Operate
   ```

2. Update Memory Promotion to distinguish:
   - knowledge log as evidence memory
   - notebook as learning debt
   - plan-of-record as accepted truth
   - friction as an Observe subcase
3. Update references that mention `sharedState` to explain internal JSON vs agent-facing "team working state".
4. Update API and CLI docs examples to use final kind names.
5. Register this plan and any new/renamed docs in `manifest.json`.
6. Run docs/search checks for obsolete terms.

Acceptance:

- No docs instruct agents to use generic "shared state" as the mental model.
- Internal schema docs can mention `sharedState` only as the JSON field name.

### Phase 7 - Full Validation and Cleanup

Owner profile: validation worker.

Commands:

```bash
cd scenarios/prompt-manager/api && go test ./...
cd scenarios/prompt-manager && prompt-manager team contract-validate --all
rg 'Durable State|rolling-artifact|append-only-log|operator-state|decision-stream|handoff-history' scenarios/prompt-manager
rg 'shared state|Shared State' scenarios/prompt-manager/store/agents scenarios/prompt-manager/store/teams scenarios/prompt-manager/docs
```

If UI code is affected:

```bash
cd scenarios/prompt-manager/ui && pnpm test -- --runInBand
```

If docs tooling is available:

```bash
knowledge-observatory docs audit scenarios/prompt-manager
```

Expected grep results:

- `Durable State`: no matches outside historical plan text or this implementation plan.
- old kind values: no matches outside this implementation plan.
- `sharedState`: allowed in schemas/code/API references as JSON field name.
- `shared state`: not allowed in agent-facing prompt prose except when explaining the old term in migration/plan docs.

## Testing Plan

### Unit Tests

Add or update tests for:

- storage kind validation
- exact universal Storage Map text
- team-specific storage rendering
- prompt section ordering
- absence of old headings
- bundled team contract validation
- bundled prompt prose drift

### Golden Prompt Tests

Add at least two focused prompt rendering tests:

1. Team with plan-of-record docs, notebooks, and multiple team working state kinds.
2. Team with no plan-of-record or notebooks, only built-in always-available storage.

These should assert:

- exact mental model headings
- path normalization
- notebook curator/promotion context
- kind-specific use text
- absence of legacy wording

### Schema Validation

Update `team.schema.json` so `documents.sharedState[].kind` enum matches final kinds.

Validate all bundled `team.json` files through existing Go tests and, if available, JSON schema tooling.

### Prompt Prose Validation

The drift test should enforce that hand-authored prompt prose does not reintroduce contract-owned storage policy.

Suggested forbidden phrase categories:

- old headings: `Durable State`
- old mental model: `shared state` in agent-facing docs
- old kind names
- generic contract boilerplate that belongs in generated sections

Keep this test focused enough to avoid false positives on legitimate file paths like `shared/TEAM.md`.

## Rollout and Validation Checklist

Use this checklist for final acceptance:

- [ ] `Storage Map` renders in every team member heartbeat prompt.
- [ ] `Your Team Storage` renders from `operatingContract.documents`.
- [ ] `Durable State` prompt section no longer exists.
- [ ] Team working state kind validation is closed and rejects old values.
- [ ] Bundled teams use only final kind names.
- [ ] Friction capture wording exists exactly in the universal Observe section.
- [ ] Meta-optimization has explicit friction-mining ownership.
- [ ] Agent/team markdown no longer repeats generic storage contract prose.
- [ ] Documentation explains Continue/Observe/Propose/Operate.
- [ ] API/CLI references use final kind names.
- [ ] Go API tests pass.
- [ ] UI tests pass if touched.
- [ ] Documentation audit passes or known unrelated findings are documented.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Prompt gets too long | Remove duplicated prose from agent/team markdown in the same cutover. Keep generated section concise and omit empty categories. |
| Agents over-report friction | Storage Map says use the lightest durable form and raise decisions only when friction blocks work or indicates missing/broken capability. |
| Team working state remains a catch-all | Closed kind enum plus kind-specific use text. Prompt prose must say "team working state", not generic "shared state". |
| Notebooks and knowledge remain conflated | Generated section explicitly says knowledge is structured observations/evidence/friction; notebook is unresolved pattern/workaround debt. |
| Plan-of-record becomes too broad | Keep team-shared-docs-design rules: plan-of-record requires consumers or durable strategic frame. |
| Tests become brittle due exact wording | Exact wording is intentional for this cutover. If wording changes later, update this plan successor and tests together. |
| Existing teams lose capability | Classify every existing shared file into a final kind; do not delete operational surfaces unless proven stale and separately approved. |

## Non-Goals and Prohibited Patterns

Non-goals:

- API/CLI enforcement of write rules.
- New storage backend.
- UI redesign.
- New team creation.
- New notebook promotion decision tree beyond friction routing.

Prohibited:

- Legacy kind aliases.
- Dual old/new prompt sections.
- Generic "shared docs" language that blurs plan-of-record and notebook.
- Generic "shared state" as agent-facing ontology.
- Duplicated storage instructions in every member file.
- New hand-authored per-team storage prose that duplicates generated `Your Team Storage`.

## Suggested Multi-Agent Work Split

These can run in parallel after Phase 1 contracts are agreed, but avoid editing the same files concurrently.

1. Contract worker:
   - Owns `api/teamcontract`, schema, bundled contract tests.
2. Prompt builder worker:
   - Owns `api/heartbeat/prompt_builder.go` and heartbeat tests.
3. Team data worker:
   - Owns bundled `store/teams/*/team.json` kind classifications.
4. Prompt prose worker:
   - Owns `store/agents/*` and `store/teams/*/{shared,members}` markdown cleanup.
5. Meta-optimization worker:
   - Owns friction topic and meta member updates.
6. Documentation worker:
   - Owns docs/reference/concepts/README/manifest updates.
7. Validation worker:
   - Owns final test run, grep audit, and any missed obsolete wording cleanup.

Coordination rule: workers must not revert unrelated dirty files. The repo may already contain unrelated UI or plan changes.

## Definition of Done

The cutover is complete when:

- The only agent-facing storage ontology is Continue / Observe / Propose / Operate.
- Every team member prompt receives the exact universal Storage Map wording or an intentionally updated successor.
- Every team member prompt receives a generated team-specific storage map from the operating contract.
- All old team working state kind names are removed from active data and validation.
- Prompt prose is shorter and domain-specific, with generic storage policy generated centrally.
- Friction capture is part of Observe and meta-optimization has explicit mining ownership.
- Tests and docs validate the new model.
- No compatibility or legacy code paths remain.
