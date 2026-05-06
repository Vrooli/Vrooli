# Prompt Manager Operating Graph Validator Refactor Plan

## Purpose

Refactor the first-pass operating graph validator into a clean contract-validation architecture before adding more contract completeness stages.

This plan is for the upfront architecture work only. The end state should make operating graph checks easy to extend, easy to test, and hard to accidentally bypass. The current behavior should remain externally recognizable for existing CLI/API users, but the internal implementation should be a hard cutover: no legacy validator path, no compatibility shim, no duplicate relationship engine, and no dead transitional code.

## Required Reading

Run this command before implementation:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health screaming-architecture-audit boundary-of-responsibility-enforcement decision-boundary-extraction seam-discovery-and-enforcement test utils-unification
```

Then read these source files:

```bash
docs/agent-system/OPERATING_GRAPHS.md
docs/marketing/OPERATING_MODEL.md
docs/plans/prompt-manager-operating-graph-contract-implementation-plan.md
scenarios/prompt-manager/api/memberflow/operating_graph.go
scenarios/prompt-manager/api/memberflow/operating_graph_validate.go
scenarios/prompt-manager/api/memberflow/operating_graph_test.go
scenarios/prompt-manager/api/memberflow/schema.go
scenarios/prompt-manager/api/memberflow/team_contracts.go
scenarios/prompt-manager/api/memberflow/handlers.go
scenarios/prompt-manager/cli/graph/operating_model.go
scenarios/prompt-manager/cli/graph/operating_model_test.go
```

Baseline commands:

```bash
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/api && go test ./memberflow
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/cli && go test ./graph
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model validate --team marketing-crew --json
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model diff --team marketing-crew --json
```

Observed baseline on 2026-05-06:

- `go test ./memberflow` passes from `scenarios/prompt-manager/api`.
- `go test ./graph` passes from `scenarios/prompt-manager/cli`.
- `prompt-manager graph operating-model validate --team marketing-crew --json` fails with the four expected `marketing/notebook/* -> member` runtime-missing errors.
- The first attempted package test from `scenarios/prompt-manager` failed because `api` and `cli` are separate Go modules. Use the module roots above.

## Greenfield Hard-Cut Rule

This scenario is still greenfield. The refactor must leave the operating graph implementation in a materially cleaner state, not merely wrap the current code.

Hard rules:

- Delete the old monolithic validation path after replacing it.
- Do not keep compatibility aliases, duplicate helper copies, fallback validators, `_old` files, `_legacy` files, or dead transitional branches.
- Do not support undocumented Mermaid syntax while refactoring.
- Do not weaken current validation to make marketing quieter.
- Do not introduce a second relationship model for diff versus validate.
- Do not add new completeness stages until the rule architecture is in place and the existing baseline rule set is re-expressed through it.

External CLI/API commands may keep their current names and response shapes because those are the product surface, not legacy internals.

## Problem Statement

The first-pass implementation proved the operating graph concept, but the validator architecture is not ready for staged contract completeness.

Current issues:

- `scenarios/prompt-manager/api/memberflow/operating_graph_validate.go` is about 915 lines and mixes runtime loading, relationship normalization, graph matching, validation rules, diff generation, suggestions, source formatting, and completeness checks.
- Current completeness checks are hard-coded in `validateOperatingGraphCompleteness` and only cover `intake[]` and `output[]`.
- `validate` and `diff` share some relationship helpers, but their responsibilities are not clearly separated. This makes future checks likely to sprawl across multiple unrelated functions.
- Rule identity, default severity, applicability by graph mode, source attribution, and suggested remediation are implicit conventions rather than first-class concepts.
- Adding a staged rule such as `decisions_owned[] must appear in graph` would currently require editing several distant functions and tests, which is the wrong direction for a contract system that should grow.

The next implementation should establish the validation architecture first, then re-add the existing rules on top of it.

## Scope

In scope:

- Refactor operating graph validation and diff internals under `scenarios/prompt-manager/api/memberflow`.
- Preserve current parser behavior unless a parser cleanup is directly needed for clearer boundaries.
- Preserve existing API and CLI command surfaces:
  - `GET /operating-graphs`
  - `GET /operating-graphs/validate`
  - `GET /operating-graphs/diff`
  - `prompt-manager graph operating-model list`
  - `prompt-manager graph operating-model validate`
  - `prompt-manager graph operating-model diff`
- Re-express all existing validation behavior through a rule registry.
- Add architecture-focused tests for rule registration, relationship indexing, validate/diff parity, and marketing fixture behavior.
- Update `docs/agent-system/OPERATING_GRAPHS.md` if rule concepts or validation stages become user-visible.

Out of scope:

- Fixing marketing graph/config mismatches.
- Adding the next completeness stages during the refactor.
- Implementing `sync --apply`.
- Validating Topic Catalog or Decisions markdown tables.
- Changing generated heartbeat `# Topic Contract` content.
- Reworking the CLI UX beyond preserving current output.

## Current Technical Context

Current operating graph implementation:

- `operating_graph.go` owns metadata extraction, Mermaid parsing, core graph DTOs, validation response DTOs, and diff DTOs.
- `operating_graph_validate.go` owns runtime relationship construction, graph relationship construction, relationship matching, diff building, suggestion text, entity validation, edge validation, and completeness validation.
- `handlers.go` wires API handlers to `LoadOperatingGraphBlocks`, `BuildOperatingGraphRuntime`, `ValidateOperatingGraphs`, and `DiffOperatingGraphs`.
- CLI code in `cli/graph/operating_model.go` is mostly a thin HTTP client and output formatter.

Current validation rules from `docs/agent-system/OPERATING_GRAPHS.md`:

- `graph_unknown_member`
- `graph_unknown_decision`
- `graph_unknown_team`
- `graph_unknown_por`
- `graph_untyped_node`
- `graph_topic_unresolved`
- `graph_future_topic_live_edge`
- `graph_edge_unbacked`
- `graph_declared_intake_missing`
- `graph_declared_output_missing`

Current diff relationship kinds:

- `topic_read`
- `topic_output`
- `por_output`
- `decision_owned`
- `decision_consumed`
- `capability_gap_raised`
- `external_producer`
- `external_producer_intake`
- `cross_team_output`

Current completeness behavior:

- Contract-mode `validate` checks all declared `intake[]` appear as `topic -> member`.
- Contract-mode `validate` checks all declared `output[]` appear as `member -> topic` or `member -> por`.
- Contract-mode `validate` does not yet enforce declared `required_read[]`, `evidence_consumed[]`, `decisions_owned[]`, `decisions_consumed[]`, `raises_capability_gaps`, or `external_producers[]` as completeness rules.
- `diff` already detects more drift than `validate`, including runtime relationships missing from the graph.

## Target End State

The operating graph subsystem should have a clear internal architecture:

```text
Markdown graph blocks
        |
        v
Mermaid parser and block extractor
        |
        v
Normalized graph/runtime contract model
        |
        v
Relationship indexes and matchers
        |
        +--> validation rule registry
        |
        +--> diff reconciliation builder
        |
        v
API/CLI response DTOs
```

The code should make the domain obvious:

- Parsing code speaks in graph blocks, metadata, typed nodes, and Mermaid subset.
- Contract model code speaks in graph relationships, runtime relationships, source references, and indexes.
- Rule code speaks in rule ids, severities, graph modes, and findings.
- Diff code speaks in drift direction, acceptable fields, suggestions, and reconciliation detail.
- API/CLI code remains orchestration and presentation only.

Adding a future completeness rule should require:

1. adding one rule file or one small rule object;
2. registering it in one rule registry;
3. adding focused tests for that rule;
4. updating docs if the rule is user-visible.

It should not require editing parser code, diff matching code, API handlers, CLI code, or unrelated rule implementations.

## Proposed File Layout

Use the existing `memberflow` package, but split files by domain responsibility. Exact names can vary slightly, but the final structure should preserve these boundaries.

```text
scenarios/prompt-manager/api/memberflow/
  operating_graph_types.go             # DTOs and stable public response types
  operating_graph_extract.go           # markdown metadata block discovery
  operating_graph_mermaid.go           # Mermaid subset parser and typed node parsing
  operating_graph_runtime.go           # runtime loading facade for operating graphs
  operating_graph_relationship.go      # relationship types, keys, match semantics
  operating_graph_contract_index.go    # normalized per-block graph/runtime indexes
  operating_graph_rules.go             # rule interfaces, registry, runner
  operating_graph_rules_entities.go    # member/topic/decision/team/por/node-kind rules
  operating_graph_rules_edges.go       # direct edge truth rules
  operating_graph_rules_completeness.go # baseline intake/output completeness rules
  operating_graph_diff.go              # diff builder and remediation suggestions
  operating_graph_findings.go          # finding construction, sorting, severity counts
```

Delete or shrink the current files after the split:

- `operating_graph.go` should not remain a mixed parser/types file if the above split is used.
- `operating_graph_validate.go` should not remain as a large procedural validator. It should be deleted or reduced to a thin compatibility-free public entrypoint that calls the new validator runner.

The public entrypoints may remain:

- `LoadOperatingGraphBlocks`
- `ExtractOperatingGraphBlocks`
- `ParseOperatingMermaid`
- `BuildOperatingGraphRuntime`
- `ValidateOperatingGraphs`
- `DiffOperatingGraphs`

Keeping those names is acceptable because handlers and tests already use them as the current API inside the package. The implementation behind them should be new, not a wrapper around the old code.

## Core Design

### Contract Context

Create a first-class context object for one graph block:

```go
type OperatingGraphContractContext struct {
    Block      OperatingGraphBlock
    Runtime    OperatingGraphRuntime
    GraphIndex OperatingGraphContractIndex
}
```

The context is built once per block and reused by validation and diff.

### Relationship Model

Keep one canonical relationship type for graph and runtime facts. Replace ad hoc scans with indexed relationship sets.

```go
type OperatingRelationship struct {
    Kind       OperatingRelationshipKind
    Team       string
    Member     string
    Topic      string
    Decision   string
    Path       string
    External   string
    TargetTeam string
    Source     OperatingSourceRef
}

type OperatingSourceRef struct {
    Path string
    Line int
}
```

Use a typed relationship kind rather than raw string constants where practical:

```go
type OperatingRelationshipKind string
```

The matcher should be centralized:

```go
type OperatingRelationshipMatcher struct{}

func (m OperatingRelationshipMatcher) GraphBackedByRuntime(graphRel OperatingRelationship, runtime OperatingRelationshipSet) bool
func (m OperatingRelationshipMatcher) RuntimeShownInGraph(runtimeRel OperatingRelationship, graph OperatingRelationshipSet) bool
```

Topic prefix overlap should still use the existing `Overlap` behavior; do not clone prefix matching.

### Relationship Index

Create an index with explicit query methods:

```go
type OperatingGraphContractIndex struct {
    NodesByID        map[string]OperatingGraphNode
    NodeIDByKindValue map[string]string
    GraphRelationships OperatingRelationshipSet
    RuntimeRelationships OperatingRelationshipSet
}
```

Required query methods:

- `Node(kind, value string) (OperatingGraphNode, bool)`
- `GraphHasRelationship(rel OperatingRelationship) bool`
- `RuntimeHasRelationship(rel OperatingRelationship) bool`
- `RuntimeRelationshipsByMember(member string) []OperatingRelationship`
- `RuntimeRelationshipsByKind(kind OperatingRelationshipKind) []OperatingRelationship`

The rules should depend on these methods, not on raw slices and nested loops.

### Rule Interface

Use an explicit rule interface:

```go
type OperatingGraphRule interface {
    ID() string
    DefaultSeverity() Severity
    AppliesTo(mode string) bool
    Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding
}
```

`OperatingGraphRuleContext` should expose only the things rules need:

```go
type OperatingGraphRuleContext struct {
    Block   OperatingGraphBlock
    Runtime OperatingGraphRuntime
    Index   OperatingGraphContractIndex
    Matcher OperatingRelationshipMatcher
}
```

Use a registry function:

```go
func DefaultOperatingGraphRules() []OperatingGraphRule
```

No global mutable registry. Keep registration deterministic and easy to test.

### Rule Groups

Add rule group metadata now, even if only baseline rules are enabled.

```go
type OperatingGraphRuleGroup string

const (
    OperatingRuleGroupEntity       OperatingGraphRuleGroup = "entity"
    OperatingRuleGroupEdgeTruth    OperatingGraphRuleGroup = "edge_truth"
    OperatingRuleGroupCompleteness OperatingGraphRuleGroup = "completeness"
)
```

Each rule can expose `Group() OperatingGraphRuleGroup` if useful for tests and future staged rollouts.

Do not add user-facing flags for rule groups in this refactor. The point is internal maintainability first. User-facing staged enforcement can come later.

### Finding Construction

Centralize finding construction so source paths, line numbers, rule ids, severity counts, and sorting are consistent.

Recommended helpers:

```go
type OperatingFindingBuilder struct {
    GraphID string
    Team    string
    RuleID  string
    Severity Severity
}

func (b OperatingFindingBuilder) WithNode(node OperatingGraphNode, detail string) OperatingGraphFinding
func (b OperatingFindingBuilder) WithEdge(edge OperatingGraphEdge, detail string) OperatingGraphFinding
func (b OperatingFindingBuilder) WithRelationship(rel OperatingRelationship, detail string) OperatingGraphFinding
```

Keep response JSON shape stable unless there is a clear reason to improve it.

### Diff Builder

Diff should use the same `OperatingGraphContractIndex` and matcher as validation.

Expected split:

- `BuildGraphOperatingRelationships` and `BuildRuntimeOperatingRelationships` become lower-level relationship builders used by the index.
- `DiffOperatingGraphs` builds contexts and calls a focused diff builder.
- Suggestion text stays in `operating_graph_diff.go`.

Do not let diff have its own relationship interpretation.

## Rule Re-Expression Plan

Rebuild the current rules on the new architecture before adding anything new.

### Entity Rules

Rules:

- `graph_untyped_node`
- `graph_unknown_node_kind`
- `graph_unknown_member`
- `graph_unknown_decision`
- `graph_unknown_team`
- `graph_unknown_por`
- `graph_topic_unresolved`

Implementation notes:

- Entity rules should iterate typed graph nodes.
- `member:` lookup uses the loaded team contract for `block.Metadata.Team`.
- `decision:` lookup uses `runtime.Contracts.HasDecisionContext`.
- `topic:` live resolution uses runtime relationship declarations for the same team.
- Future/old/external topic qualifiers remain excluded from live topic resolution.

### Edge Truth Rules

Rules:

- `graph_future_topic_live_edge`
- `graph_edge_unbacked`

Implementation notes:

- The rule should skip process/future grouping nodes as current behavior does.
- Future topic on an active direct edge remains warning.
- Backing check should use canonical relationship matching through the index.

### Baseline Completeness Rules

Rules:

- `graph_declared_intake_missing`
- `graph_declared_output_missing`

Implementation notes:

- `graph_declared_intake_missing` remains error.
- `graph_declared_output_missing` remains warning.
- Use runtime relationships as the source of truth, not direct iteration over `Topics` inside the rule. This keeps future completeness checks uniform.
- Do not add required-read/evidence/decision/external/capability completeness in this refactor.

## Implementation Phases

### Phase 0: Baseline Snapshot

Record current behavior before moving code:

```bash
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/api && go test ./memberflow
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/cli && go test ./graph
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model validate --team marketing-crew --json > /tmp/marketing-operating-validate.before.json || true
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model diff --team marketing-crew --json > /tmp/marketing-operating-diff.before.json
```

Acceptance:

- The snapshot shows the four known notebook validation errors.
- API and CLI focused tests pass from their module roots.

### Phase 1: Split Parser and DTO Responsibilities

Move code from `operating_graph.go` into responsibility-specific files.

Deliverables:

- DTOs live in `operating_graph_types.go`.
- Markdown block extraction lives in `operating_graph_extract.go`.
- Mermaid parser lives in `operating_graph_mermaid.go`.
- Existing parser tests still pass without test weakening.

Acceptance:

- No behavior change.
- `TestMarketingOperatingModelUsesReadableAnnotatedLabels` still proves hidden annotations keep labels readable.
- No duplicate parser functions remain.

### Phase 2: Introduce Contract Index and Relationship Set

Create the normalized contract model.

Deliverables:

- `OperatingRelationshipKind`
- `OperatingSourceRef`
- `OperatingRelationshipSet`
- `OperatingGraphContractIndex`
- centralized matcher
- graph relationship builder
- runtime relationship builder

Acceptance:

- Existing diff tests pass through the new relationship set.
- Add unit tests for:
  - graph `topic -> member` matches runtime intake, required read, and evidence consumed;
  - runtime required read missing from graph is detected by diff;
  - future topic relationships are omitted from actionable graph relationships;
  - cross-team outputs and PoR outputs still match.

### Phase 3: Add Rule Runner and Re-Express Existing Rules

Replace procedural validation with a rule registry.

Deliverables:

- `OperatingGraphRule`
- `OperatingGraphRuleContext`
- `DefaultOperatingGraphRules`
- entity rule implementations
- edge truth rule implementations
- baseline completeness rule implementations
- centralized finding sorting and severity counting

Acceptance:

- `ValidateOperatingGraphs` is a thin runner over contexts and rules.
- Current validation response for marketing still reports the same four notebook errors and no new rule class.
- Existing validation unit tests are updated to assert rule-specific behavior through the new runner.
- Add one test proving that rule registration contains exactly the expected baseline rule ids.

### Phase 4: Rebuild Diff on the Shared Contract Index

Make diff consume the same normalized relationship index and matcher as validation.

Deliverables:

- `operating_graph_diff.go` owns drift generation and suggestions.
- Existing `DiffOperatingGraphs` behavior is preserved.
- No relationship matching logic remains duplicated in diff.

Acceptance:

- Existing diff tests pass.
- Marketing `diff` still reports current graph/runtime drift, including the four notebook reads and runtime relationships missing from graph.
- Suggestions and acceptable fields remain present in JSON output.

### Phase 5: Handler and CLI Verification

Ensure orchestration layers did not absorb domain logic.

Deliverables:

- API handlers still only load blocks/runtime, filter, call validator/diff, and write JSON.
- CLI still only fetches JSON and formats it.
- Any new response fields are optional and intentionally documented. Prefer no response shape changes in this refactor.

Acceptance:

- `go test ./memberflow` passes from `scenarios/prompt-manager/api`.
- `go test ./graph` passes from `scenarios/prompt-manager/cli`.
- No operating graph rule logic appears in CLI files.

### Phase 6: Documentation Update

Update docs only for actual architecture or user-visible rule changes.

Deliverables:

- If rule grouping or validation architecture is user-visible, update `docs/agent-system/OPERATING_GRAPHS.md`.
- Add or adjust `[CODE: ...]` references only if the repo's current docs validation supports them for this area.
- Do not document internal file splits in user-facing prose unless they affect contributors.

Acceptance:

- Operating graph docs still match command behavior.
- No stale mention of a monolithic validator remains if docs mention internals.

### Phase 7: Final Cleanup

Perform hard-cut cleanup.

Deliverables:

- Remove old helper functions superseded by the relationship set, matcher, rule runner, or diff builder.
- Remove redundant tests that only assert implementation details of deleted functions, replacing them with rule/index tests where needed.
- Run `rg` to ensure no obsolete names remain.

Commands:

```bash
rg "validateOperatingGraphCompleteness|operatingEdgeBacked|graphlessRelationshipLooksRelated|operatingRuntimeDiffKey" scenarios/prompt-manager/api/memberflow
rg "legacy|compat|old|deprecated" scenarios/prompt-manager/api/memberflow/operating_graph*
```

Acceptance:

- Any remaining matches are intentional current names, not dead compatibility.
- No `_old`, `_legacy`, or duplicate validator files exist.

## Testing Plan

### Unit Tests

Add focused tests for:

- relationship key stability and deduplication;
- relationship matching by kind;
- topic overlap matching;
- graph future/old/external topic exclusion;
- runtime relationship extraction from every relevant `topics.json` field;
- rule applicability by graph mode;
- entity rules;
- direct edge backing rule;
- baseline intake completeness rule;
- baseline output completeness rule;
- finding sorting and severity counts.

### Integration-Style Tests

Keep and strengthen tests using the real marketing operating model:

- parser extracts one marketing contract graph;
- labels remain user-readable and machine tokens remain hidden in comments;
- validation reports the expected current notebook errors until marketing is reconciled;
- diff reports both graph-missing-runtime and runtime-missing-graph relationships.

If marketing mismatches are fixed before this refactor lands, update the fixture expectations to the new accepted state. Do not hard-code stale mismatches.

### CLI/API Tests

Add or keep tests proving:

- validate exits non-zero when errors exist;
- diff exits zero and reports drift;
- JSON output is valid and includes findings/diffs;
- human output groups diff directions correctly.

## Validation Checklist

Run:

```bash
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/api && go test ./memberflow
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/api && go test ./...
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/cli && go test ./graph
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/cli && go test ./...
cd /home/matthalloran8/Vrooli && prompt-manager graph topics --team marketing-crew --json
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model validate --team marketing-crew --json
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model diff --team marketing-crew --json
```

Expected result after refactor, before marketing design reconciliation:

- Topic graph remains clean.
- Operating model validate still reports only the known current contract/config mismatches unless marketing has been intentionally updated.
- Operating model diff remains usable and complete.
- API and CLI focused tests pass.

Full repo `make test` may still fail on unrelated existing lint/docs gates. If so, record the unrelated failures and do not hide scenario-specific regressions behind them.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| The refactor changes validation semantics accidentally. | Snapshot marketing validate/diff output before refactor and compare after. |
| Rule engine becomes too abstract. | Keep rule interface small; do not add user-facing rule config yet. |
| Relationship matching drifts between validate and diff. | Use one matcher and one relationship set for both. |
| Tests overfit internals. | Test rules through stable contexts and relationship facts, not private loop structure. |
| File split becomes cosmetic. | Delete old monolith code and verify each new file has one clear responsibility. |
| Future completeness stages still require broad edits. | Add a small fake completeness rule in a test or compile-time example to prove one-place registration. |

## Non-Goals and Prohibited Patterns

Non-goals:

- No marketing model decisions.
- No new validator stages beyond re-expressing current baseline rules.
- No sync/apply command.
- No arbitrary Mermaid parser expansion.
- No Topic Catalog or Decisions table validation.

Prohibited patterns:

- Large `switch` blocks that encode every future rule in one function.
- A validator path and a separate diff path that each infer relationship semantics independently.
- Tests that only check that a helper was called.
- Compatibility wrappers around deleted helper names.
- Silent fallbacks when graph metadata or contract syntax is invalid.
- Adding config flags before there is a proven need.

## Definition of Done

The plan is complete when:

- The old monolithic validator implementation has been replaced by a rule-driven architecture.
- Parser, contract model, rules, diff, findings, handlers, and CLI responsibilities are visibly separated.
- Current operating graph behavior is preserved for the existing baseline rule set.
- Adding the next completeness rule requires a focused rule implementation, one registry update, and focused tests.
- `go test ./memberflow` passes from `scenarios/prompt-manager/api`.
- `go test ./graph` passes from `scenarios/prompt-manager/cli`.
- Marketing operating graph validate/diff outputs are understood and not accidentally changed.
- No legacy, compatibility, duplicate, or dead operating graph validator code remains.

## Follow-Up After This Plan

After this refactor lands, add completeness stages one at a time:

1. decision ownership completeness;
2. decision consumption completeness;
3. required-read completeness;
4. evidence-consumed completeness;
5. capability-gap completeness;
6. external-producer completeness.

Each stage should be implemented as a new rule or small rule family, validated against marketing, and reviewed as a design decision before promoting warnings to errors.
