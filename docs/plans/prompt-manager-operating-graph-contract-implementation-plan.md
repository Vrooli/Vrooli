# Prompt Manager Operating Graph Contract Implementation Plan

## Purpose

Implement a first-class operating-graph contract layer for prompt-manager teams, starting with `marketing-crew`. The feature lets humans and agents reason in plan-of-record Mermaid diagrams while making those diagrams accountable to runtime declarations, generated heartbeat prompts, and validation output.

This plan covers the full implementation path for:

- typed Mermaid operating-model diagrams in plan-of-record docs;
- parsing those diagrams into a normalized graph model;
- validating the normalized graph against `team.json`, member `topics.json`, taxonomies, decision contexts, PoR files, and prompt previews;
- adding generated member-specific `# Topic Contract` heartbeat sections;
- applying and validating the pattern against `docs/marketing/OPERATING_MODEL.md`.

## Required Reading

Run these before implementation:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Then read:

```bash
docs/agent-system/TOPICS_SCHEMA.md
docs/agent-system/INTAKE_PIPELINE.md
docs/agent-system/PROSE_SCAN_TARGETS.md
docs/reference/machine-readable-references.md
docs/marketing/OPERATING_MODEL.md
scenarios/prompt-manager/api/memberflow/schema.go
scenarios/prompt-manager/api/memberflow/validation.go
scenarios/prompt-manager/api/memberflow/handlers.go
scenarios/prompt-manager/api/heartbeat/prompt_builder.go
scenarios/prompt-manager/api/heartbeat/inbox_flow.go
scenarios/prompt-manager/cli/graph/graph.go
scenarios/prompt-manager/cli/graph/topics.go
```

Useful baseline commands:

```bash
cd scenarios/prompt-manager/cli
go run . graph topics --team marketing-crew --json
go run . team prompt-matrix marketing-crew --json
```

Observed baseline on 2026-05-06: `graph topics --team marketing-crew --json` returned `0 errors`, `0 warnings`.

## Greenfield Constraint

This is greenfield work. Do not include compatibility shims, legacy wrappers, dead code, deprecated aliases, `_unused` variables, broad fallback parsers, or migration-only branches.

For this feature:

- Do not support arbitrary Mermaid syntax. Support only the documented subset.
- Do not keep old untyped contract diagrams as a second contract source.
- Do not add "best effort" runtime behavior from parsed Mermaid. Validation can fail; runtime prompts remain generated from structured declarations.
- Do not add sync auto-apply in the first implementation. Build validate/diff first.
- Do not weaken existing topic validation rules to make the new graph layer quiet.

## Problem Statement

Prompt-manager currently has three partially connected layers:

1. Human plan-of-record docs, including Mermaid operating diagrams.
2. Machine-readable runtime declarations in `team.json` and member `topics.json`.
3. Generated heartbeat prompts that agents actually receive.

Marketing now has a coherent operating model, but its full topic-level Mermaid diagram is only documentation. It is not parsed, validated, or connected to the generated prompts. Meanwhile, `topics.json` drives some prompt content, but only partially:

- `required_read[]` appears in `# Active Task Brief` under `## Required Memory`.
- `intake[]` members get a generated `# Inbox Flow`.
- non-intake members do not get a generated full topic contract.
- output, evidence, and decision relationships mostly rely on prose in `RESPONSIBILITIES.md`, `HEARTBEAT.md`, and the operating model.

The new system must make the relationship explicit:

```text
PoR Mermaid diagram -> normalized operating graph -> validation against runtime config -> generated prompt contracts -> prompt preview validation
```

## Scope

In scope:

- Define the canonical operating-graph documentation and typed Mermaid subset.
- Add a parser/extractor for marked Mermaid graph blocks.
- Add normalized operating graph types.
- Add graph-contract validation against the live prompt-manager declarations.
- Add CLI/API surfaces to validate and diff operating graphs.
- Add generated `# Topic Contract` sections to heartbeat prompts for every team member.
- Add tests for parser, validator, CLI output, prompt rendering, and marketing fixture behavior.
- Type the marketing full topic-level diagram and validate it.
- Report marketing graph/config/doc mismatches surfaced by the new validator.

Out of scope:

- Automatically applying sync changes.
- Making Mermaid the runtime source of truth.
- Generalizing to every existing Mermaid diagram in the repo.
- Reworking marketing team behavior before reviewing new validation findings.
- Merging or renaming marketing members/topics as part of this implementation.
- Adding UI graph editing.

## Current Technical Context

Existing topic declarations:

- `scenarios/prompt-manager/api/memberflow/schema.go` defines `Topics` with `intake`, `required_read`, `evidence_consumed`, `output`, `decisions_owned`, `decisions_consumed`, `raises_capability_gaps`, and `external_producers`.
- `scenarios/prompt-manager/api/memberflow/validation.go` validates the declared topic graph.
- `scenarios/prompt-manager/api/memberflow/handlers.go` exposes `/topics/graph`, building a graph from `topics.json`.
- `scenarios/prompt-manager/cli/graph/topics.go` wraps `/topics/graph` as `prompt-manager graph topics`.

Existing prompt generation:

- `scenarios/prompt-manager/api/heartbeat/prompt_builder.go` builds ordered structured sections.
- `# Active Task Brief` currently renders `required_read[]` under `## Required Memory`.
- `scenarios/prompt-manager/api/heartbeat/inbox_flow.go` renders `# Inbox Flow` only for members with `intake[]`.
- `team prompt-matrix` exposes structured prompt sections for all team members.

Existing docs and markers:

- `docs/reference/machine-readable-references.md` defines inline markers such as `topic:`, `topic[future]:`, `member` is not currently a marker, and `team:`/`agent:`/`decision:` exist.
- `packages/api-core/markedrefs` owns marker parsing and qualifier behavior.
- `docs/marketing/OPERATING_MODEL.md` contains the marketing compact diagram and full topic-level diagram.

Important existing shape:

- `prompt-manager graph topics --team marketing-crew --json` is clean today.
- The marketing diagram has conceptual nodes such as `Planning decisions`, `Marketing canon`, `Learning synthesis`, and `Operator approval`; the new validator must either type these as `process:`/`external:`/`por:` nodes or require the diagram to remove/replace them.
- Marketing uses future/transition topics such as `topic[future]:ad-run/*`, `topic[future]:publish-performance/*`, and JSONL-backed `published-scenario-mentions`.

## Target End State

1. A canonical doc explains operating-graph contract syntax and semantics.
2. Contract graphs use metadata blocks:

   ````markdown
   <!-- prompt-manager-graph:
   id: marketing-operating-model
   scope: team
   team: marketing-crew
   mode: contract
   -->
   ```mermaid
   flowchart LR
     R["member:researcher<br/>Researcher"]
     RI["topic:research-inbox/*"]
     RI --> R
   ```
   ````

3. The parser extracts a normalized graph model from marked Mermaid blocks.
4. The validator compares normalized graph edges to runtime declarations.
5. New CLI/API commands validate and diff operating graphs.
6. Every heartbeat prompt has a generated `# Topic Contract` section from `topics.json`.
7. Prompt preview tests prove generated topic contracts match member declarations.
8. The marketing operating model is converted to typed contract syntax and validated.
9. Any remaining marketing mismatch is reported as an explicit graph/config/design decision, not hidden in prose.

## Contract Decisions

### D1. Authoring Model

Use typed Mermaid as the design source, structured config as the runtime source, and validators as the bridge.

Do not generate the Mermaid diagram from JSON in the first pass. Do not execute runtime behavior from Mermaid.

### D2. Diagram Modes

Support three modes:

| Mode | Meaning | Validation |
|---|---|---|
| `explanatory` | Conceptual diagram | parse only, no contract checks |
| `checkable` | Partial diagram with typed nodes | validate typed entities present; no completeness checks |
| `contract` | Team operating contract | strict graph/config/prompt agreement checks |

Only `contract` mode is in scope for marketing adoption.

### D3. Supported Node Kinds

Initial supported kinds:

| Kind | Example | Validation source |
|---|---|---|
| `member` | `member:researcher` | team active members / `team.json::operatingContract.members` |
| `topic` | `topic:campaign-draft/*` | member `topics.json` and team `knowledgeTopics` |
| `decision` | `decision:content-publish-proposal` | `team.json::operatingContract.decisionContexts` |
| `por` | `por:docs/marketing/STRATEGY.md` | filesystem + team PoR docs |
| `external` | `external:operator` | `external_producers`, explicit allowed external nodes, or marked external |
| `team` | `team:monetization` | store team registry |
| `process` | `process:learning-synthesis` | allowed only as non-runtime grouping node |
| `future` | `future:publish-telemetry` | allowed target-state grouping node |

Do not use untyped display labels as validator inputs.

### D4. Topic Qualifiers

Use existing marked-reference qualifier semantics:

- `topic:` means live/current and must resolve.
- `topic[future]:` means target-state and absence is allowed.
- `topic[old]:` means historical and must not be treated as live.
- `topic[external]:` means outside this team/repo.
- `topic[example]:` is illustrative and should not appear in contract graphs.

For Mermaid labels, the parser should accept the same syntax inside node labels, e.g. `ARUN["topic[future]:ad-run/<lane>/*"]`.

### D5. Edge Semantics

For contract graphs:

| Edge | Required runtime match |
|---|---|
| `external -> topic` | topic is an intake prefix and consumer member declares matching `external_producers`, or external is marked allowed |
| `topic -> member` | member declares matching `intake[]`, `required_read[]`, or `evidence_consumed[]` |
| `member -> topic` | member declares matching `output[]` |
| `member -> decision` | member declares `decisions_owned[]` or `raises_capability_gaps` for `capability-gap` |
| `decision -> member` | member declares `decisions_consumed[]` or evidence relationship |
| `member -> por` | member output includes `destination_kind=por_file` for that path, or member owns a decision whose accepted form edits that PoR |
| `topic -> team` | matching cross-team output destination exists |
| `process` edges | allowed only for readability; must not satisfy runtime completeness |

In the first pass, edge labels are optional and ignored. The validator infers relationship from typed node kinds and direction.

### D6. Prompt Contract Placement

Add `# Topic Contract` after `# Operating Policy` and before `# Inbox Flow`.

Rationale: operating policy says who the member is and its storage/writes; topic contract says its runtime flow; inbox flow gives detailed drain mechanics for members with inboxes.

### D7. Sync Scope

First implementation includes `validate` and `diff`, not `sync --apply`.

`sync --dry-run` can be added only after validation is stable and marketing has produced at least one useful drift report.

## Implementation Strategy

### Phase 1: Canonical Graph Contract Doc

Create `docs/agent-system/OPERATING_GRAPHS.md`.

Content requirements:

- purpose and relationship to `TOPICS_SCHEMA.md`;
- metadata block schema;
- mode semantics;
- typed node syntax;
- supported Mermaid subset;
- edge semantics;
- qualifier behavior;
- validation severity model;
- contract graph adoption checklist;
- non-goals and prohibited patterns.

Update `docs/agent-system/README.md` file table to include the new doc.

Validation:

```bash
rg "OPERATING_GRAPHS" docs/agent-system
```

### Phase 2: Normalized Graph Package

Add a focused package under:

```text
scenarios/prompt-manager/api/memberflow/operatinggraph/
```

or, if package boundaries become awkward, under:

```text
scenarios/prompt-manager/api/memberflow/
```

Preferred files if using a subpackage:

```text
metadata.go        // metadata block parsing
mermaid.go         // documented Mermaid subset parser
model.go           // Graph, Node, Edge, SourceLocation
normalize.go       // label parsing and typed entity normalization
validate.go        // graph-contract validation
diff.go            // runtime-vs-contract diff helpers
docscan.go         // find marked graph blocks in docs
*_test.go
testdata/
```

Keep parser logic pure and deterministic. File walking belongs in `docscan.go`; validation should accept already-loaded dependencies so tests can inject fixtures.

### Phase 3: Metadata And Mermaid Extraction

Implement:

- scan Markdown for `<!-- prompt-manager-graph: ... -->` followed by a Mermaid code fence;
- parse metadata fields:
  - `id`
  - `scope`
  - `team`
  - `mode`
  - optional `status`
  - optional `allow`
- reject malformed metadata for `contract` mode;
- support one graph block per metadata comment;
- capture file and line numbers for findings.

Supported Mermaid subset:

- `flowchart LR|TB|RL|BT`
- comments beginning `%%`
- node declarations:
  - `ID[label]`
  - `ID["label"]`
  - `ID[Label<br/>Display]`
- simple directed edges:
  - `A --> B`
  - `A -->|label| B` may parse but label can be ignored initially
- chained edge fanout is not required in phase 1.

Fail fast on unsupported syntax in `contract` mode. For `explanatory`, report parser warnings without failing runtime validation.

### Phase 4: Typed Entity Normalization

Implement label normalization:

- For labels with `<br/>`, use the first line as the machine token.
- Trim quotes and HTML entity escapes where needed.
- Parse `topic[...]`, `topic:`, `member:`, `decision:`, `por:`, `external:`, `team:`, `process:`, and `future:`.
- Preserve display text separately for human output.
- Preserve qualifiers for validation decisions.

Node IDs are only local Mermaid identifiers; do not encode semantics in IDs.

### Phase 5: Runtime Indexes

Build reusable runtime indexes from existing declarations:

- team member index from store team config;
- topic input/read/evidence/output index from `memberflow.LoadTeam` or `LoadAll`;
- decision context index from `LoadAllTeamContracts`;
- taxonomy index from `LoadAllTaxonomies`;
- PoR path index from `team.json::operatingContract.documents.planOfRecord`;
- prompt-section index from `team prompt-matrix` equivalent in API tests or `heartbeat.BuildStructured`.

Keep this indexing code separate from the Mermaid parser. It is the seam that makes tests small.

### Phase 6: Graph Contract Validator

Implement validation rules with stable rule IDs.

Initial required rules:

| Rule | Severity in `contract` | Meaning |
|---|---|---|
| `graph_unknown_member` | error | `member:` node not in team contract |
| `graph_unknown_decision` | error | `decision:` node not in `decisionContexts` |
| `graph_unknown_team` | warning | `team:` node not found |
| `graph_unknown_por` | error | `por:` node path missing or not a declared PoR surface when required |
| `graph_untyped_node` | error | contract graph node lacks typed machine label |
| `graph_topic_unresolved` | error | live `topic:` node has no matching runtime topic relationship |
| `graph_future_topic_live_edge` | warning | future topic is used as if it were live runtime |
| `graph_edge_unbacked` | error | graph edge implies relationship absent from config |
| `graph_declared_output_missing` | warning initially | live output in `topics.json` missing from contract graph |
| `graph_declared_intake_missing` | error | live intake missing from contract graph |
| `graph_prompt_contract_missing` | error | prompt preview lacks `topic-contract` section |
| `graph_prompt_contract_drift` | error | prompt contract content does not match `topics.json` |

Start completeness rules at warning except live intake and prompt contract presence. Promote only after marketing is reconciled.

### Phase 7: API Endpoints

Add memberflow endpoints:

```text
GET /operating-graphs
GET /operating-graphs/validate?team=<team>&id=<id>
GET /operating-graphs/diff?team=<team>&id=<id>
```

Response shapes should be stable and compact:

```json
{
  "graphs": [],
  "validation": {
    "findings": [],
    "errors": 0,
    "warnings": 0
  }
}
```

Keep business logic out of HTTP handlers. Handlers should load store/doc context, call the graph package, and serialize the response.

Update route registration in `scenarios/prompt-manager/api/main.go`.

Update `scenarios/prompt-manager/cli/parity/coverage.json` with precise CLI coverage entries.

### Phase 8: CLI Commands

Extend `prompt-manager graph`:

```bash
prompt-manager graph operating-model list [--json]
prompt-manager graph operating-model validate --team marketing-crew [--id marketing-operating-model] [--json]
prompt-manager graph operating-model diff --team marketing-crew [--id marketing-operating-model] [--json]
```

Human output should follow operational output contract:

```text
Status
Triage
Next Steps
```

Exit code:

- `validate` returns non-zero when error findings exist.
- `diff` returns zero unless `--strict` is added later. Do not add `--strict` in phase 1 unless needed.

### Phase 9: Generated Topic Contract Prompt Section

Add:

- new prompt constants in `prompt_templates.go`:
  - kind: `topic-contract`
  - label: `Topic Contract`
  - heading: `# Topic Contract`
- new renderer, probably `topic_contract.go`;
- pure renderer input from `memberflow.MemberTopics`;
- prompt builder insertion after `# Operating Policy` and before `# Inbox Flow`.

Suggested content:

```markdown
# Topic Contract

This section is generated from `topics.json`. It is the source of truth for topic reads, writes, decisions, and capability-gap routing.

## Inboxes Drained
- `research-inbox/*` - taxonomy `marketing-research`, classifier `marketing-signal-classifier`

## Required Reads
- `audience-scan/*`

## Evidence Consumed
- `challenge-report/*` - for `content-publish-proposal`, `coverage-gap`

## Outputs
- `campaign-draft/*` - knowledge
- `oss-ad-run/*` - knowledge

## Decisions
- own/propose: `content-publish-proposal`, `coverage-gap`
- consume: `capability-gap`
- may raise `capability-gap`: yes

## External Producers
- `operator`
- `vision-walk`
```

If a member's `topics.json` is empty, render a short explicit section:

```markdown
# Topic Contract

No topic flow is declared for this member.
```

Do not duplicate the detailed drain procedure from `# Inbox Flow`.

### Phase 10: Prompt Preview Validation

Add tests that:

- every active member on every enabled team gets a `topic-contract` section;
- the section is deterministic and sorted;
- rendered rows match `topics.json`;
- `# Inbox Flow` still appears only for intake members;
- `BuildStructured` and flat `Build` remain equivalent;
- marketing prompt matrix includes `topic-contract` for all six members.

Use existing tests in `prompt_builder_test.go`, `inbox_flow_test.go`, and `handlers_prompt_matrix_test.go` as patterns.

### Phase 11: Marketing Diagram Adoption

Update only the full topic-level diagram in `docs/marketing/OPERATING_MODEL.md` to typed contract syntax.

Keep the compact diagram explanatory unless there is a strong reason to validate it.

Add metadata before the full diagram:

```markdown
<!-- prompt-manager-graph:
id: marketing-operating-model
scope: team
team: marketing-crew
mode: contract
-->
```

Expected typing decisions:

- `Researcher` -> `member:researcher`
- `Brand Manager` -> `member:brand-manager`
- `Advertiser / Draft Producer` should probably become a `process:advertiser-draft-producer` node connected to `member:oss-advertiser` and `member:subscription-advertiser`, or the graph should show both members explicitly.
- `Publisher` -> `member:publisher`
- `Marketing Contrarian` -> `member:marketing-contrarian`
- `Planning decisions` -> `process:planning-decisions`
- `Marketing canon` -> one or more `por:` nodes, or `process:marketing-canon` if intentionally summarized.
- `Operator approval` -> `external:operator-approval`
- `Learning synthesis` -> `process:learning-synthesis`
- `Future telemetry` -> `future:publish-telemetry`
- `ad-run/<lane>/*` -> `topic[future]:ad-run/<lane>/*`
- live transitional run topics remain `topic:oss-ad-run/*` and `topic:subscription-ad-run/*` if they are retained in the contract.

Do not change marketing config during this phase unless the user separately approves the resulting graph/config decisions.

### Phase 12: Validation And Reporting

Run:

```bash
cd scenarios/prompt-manager/api
go test ./memberflow ./heartbeat

cd ../cli
go test ./graph ./teams ./parity
go run . graph topics --team marketing-crew --json
go run . graph operating-model validate --team marketing-crew --json
go run . team prompt-matrix marketing-crew --json
```

Expected result:

- existing topic graph remains clean;
- operating model validator may report marketing design/config mismatches;
- prompt matrix includes `topic-contract` for every marketing member;
- no API/CLI parity stale route entries.

## Testing Plan

Unit tests:

- metadata parser accepts valid YAML-like metadata and rejects missing required fields;
- Mermaid parser accepts supported subset and rejects unsupported contract syntax;
- label normalizer handles typed labels, display labels, `<br/>`, and marked topic qualifiers;
- validator detects unknown member, unbacked topic edge, missing prompt contract, missing declared output, future topic misuse;
- topic contract renderer handles empty, intake, non-intake, evidence-only, and cross-team outputs.

Integration-ish tests:

- fixture Markdown with a contract graph validates against fixture `topics.json` and `team.json`;
- marketing real store fixture produces deterministic findings;
- prompt matrix route includes `topic-contract` sections.

CLI tests:

- `graph operating-model validate --json` calls the expected endpoint and prints JSON;
- human output groups findings by severity/remediation path;
- non-zero exit on validation errors;
- coverage parity includes the new routes.

Docs tests:

- new operating graph doc is referenced from `docs/agent-system/README.md`;
- no unqualified live `topic:` refs are introduced unless declared.

## Rollout / Validation Checklist

1. New canonical docs added and linked.
2. Parser package added with unit tests.
3. Validator package added with unit tests.
4. API endpoints added with handler tests.
5. CLI commands added with tests.
6. CLI parity map updated and passing.
7. `# Topic Contract` renderer added with tests.
8. Prompt builder order updated and covered by tests.
9. Marketing full diagram converted to typed contract graph.
10. `prompt-manager graph topics --team marketing-crew --json` remains clean.
11. `prompt-manager graph operating-model validate --team marketing-crew --json` produces expected findings.
12. Findings are reported to the user before any marketing behavior/config changes.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Mermaid parser grows into a full language parser | Document and enforce a small supported subset |
| Conceptual nodes create noisy validation | Require `process:` / `future:` kinds and make their semantics explicit |
| Prompt contract duplicates Inbox Flow | Keep Topic Contract as summary; keep Inbox Flow as drain procedure |
| Validator confuses target-state topics with live config | Use existing `topic[future]:` qualifier semantics |
| Sync applies bad changes | Do not implement auto-apply in this plan |
| Marketing graph has legitimate abstractions | Use `process:` nodes and warning-level completeness initially |
| Package dependency cycles | Keep parser/validator pure and inject runtime indexes |

## Non-Goals / Prohibited Patterns

- No direct runtime behavior from Mermaid.
- No broad support for arbitrary Mermaid.
- No compatibility aliases or duplicate old command names.
- No parser fallback that silently infers untyped nodes in `contract` mode.
- No automatic sync/apply in the first implementation.
- No unrelated marketing team redesign.
- No docs-only validation that lacks prompt-preview checks.
- No new generic `utils` dumping ground.

## Definition Of Done

The feature is done when:

- `docs/agent-system/OPERATING_GRAPHS.md` defines the contract.
- The marketing full operating diagram has contract metadata and typed nodes.
- `prompt-manager graph operating-model validate --team marketing-crew --json` exists and validates the diagram against runtime config.
- `prompt-manager graph operating-model diff --team marketing-crew --json` exists and reports contract/config differences.
- Every active prompt-manager team member receives a generated `# Topic Contract` prompt section.
- Prompt preview tests prove `# Topic Contract` content matches `topics.json`.
- Existing `prompt-manager graph topics --team marketing-crew --json` remains clean.
- New tests cover parser, validator, CLI, API, and prompt generation.
- API/CLI parity tests pass.
- The implementation introduces no legacy compatibility shims, no dead code, and no untested core parser/validator behavior.

## Final Scenario Verification

Because this modifies the `prompt-manager` scenario, finish by running:

```bash
cd scenarios/prompt-manager
make test
vrooli scenario restart prompt-manager
vrooli scenario status prompt-manager
```

If `make test` is too broad for an intermediate phase, at minimum run:

```bash
cd scenarios/prompt-manager/api && go test ./memberflow ./heartbeat ./teamcontract
cd ../cli && go test ./graph ./teams ./parity
```

Fix all lint, type, and unit test failures in modified files, including failures that appear pre-existing, unless the user explicitly narrows the verification scope.
