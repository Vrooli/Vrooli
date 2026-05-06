# Prompt Manager Operating Graph Contract Hardening Plan

## Purpose

Complete the post-refactor hardening pass for Prompt Manager operating graph contracts now that the marketing graph/config reconciliation is clean.

This plan turns the remaining review recommendations into an executable implementation sequence:

- make malformed graph metadata blocks fail predictably;
- expand contract completeness validation beyond `intake` and outputs;
- align validate/diff semantics so "contract clean" is meaningful;
- tighten team-scoped entity validation;
- clarify PoR validation semantics;
- polish generated JSON and heartbeat Topic Contract rendering;
- improve tests and docs around the operating graph contract layer.

## Required Reading

Run these first:

```bash
prompt-manager skill read documentation-health test seam-discovery-and-enforcement api-steer cli-steer boundary-of-responsibility-enforcement decision-boundary-extraction
```

Discovery evidence:

```bash
prompt-manager discover "operating graph validator completeness" "prompt-manager cli api validation" "heartbeat topic contract tests" "documentation contract drift" --complexity architectural
```

The discovery command also returned `interoperability-steer`, `ux`, and `react-coherence`; they were excluded from required reading because this pass is API/CLI/domain validation, not UI or external interoperability work.

## Greenfield Constraint

This is greenfield Prompt Manager contract infrastructure. Do not add compatibility shims, aliases, fallback validators, legacy parser paths, `_old` files, duplicate relationship models, or dead code.

If a behavior should change, cut over directly and update all call sites, tests, and docs in the same pass.

## Current Baseline

As of this plan, the reconciled marketing state is clean:

```bash
prompt-manager graph operating-model validate --team marketing-crew
# Validated 1 operating graph(s): 0 error(s), 0 warning(s).

prompt-manager graph operating-model diff --team marketing-crew
# Found 0 diff item(s).

prompt-manager graph topics --team marketing-crew --json
# 0 errors, 0 warnings
```

Focused package checks were clean in the previous review:

```bash
cd scenarios/prompt-manager/api && go test ./...
cd scenarios/prompt-manager/cli && go test ./...
```

The operating graph refactor currently lives primarily in:

- `scenarios/prompt-manager/api/memberflow/operating_graph_types.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_extract.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_mermaid.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_runtime.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_relationship.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_contract_index.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_rules*.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_diff.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_findings.go`
- `scenarios/prompt-manager/api/memberflow/operating_graph_test.go`
- `scenarios/prompt-manager/api/memberflow/handlers.go`
- `scenarios/prompt-manager/cli/graph/operating_model.go`
- `scenarios/prompt-manager/cli/graph/operating_model_test.go`
- `scenarios/prompt-manager/api/heartbeat/topic_contract.go`
- `scenarios/prompt-manager/api/heartbeat/topic_contract_test.go`
- `docs/agent-system/OPERATING_GRAPHS.md`
- `docs/marketing/OPERATING_MODEL.md`

## Problem Statement

The refactor separated responsibilities well, but the contract layer is not yet fully hardened:

1. `checkable` graph parse errors can be silently dropped because only `contract` parse failures are fatal.
2. Validation completeness currently enforces only `intake` and output visibility, while diff understands many more relationship kinds.
3. A user can see `validate` clean and `diff` dirty unless the missing relationship kinds are promoted into validation.
4. `decision:` nodes resolve against any team contract, which is too broad for a team-scoped contract unless cross-team decisions are explicit.
5. `por:` nodes only validate file existence, while docs imply stronger plan-of-record semantics.
6. Empty validation findings can serialize as `null`, which is a rough API/CLI contract.
7. `Topic Contract` evidence rendering needs an explicit behavior for evidence entries with no `for_decisions`.
8. Docs explain that tables are not validated, but do not yet describe the next completeness stages and clean-state expectations clearly enough.

## Scope

In scope:

- operating graph extraction/parser behavior;
- operating graph validation rules;
- relationship matcher/index behavior only where needed for validation parity;
- diff/validate semantic alignment;
- API response shape polish;
- CLI behavior only if API response changes require it;
- heartbeat Topic Contract rendering and tests;
- operating graph docs and marketing operating model notes;
- high-signal unit and handler tests.

Out of scope:

- changing marketing team design or topic declarations unless a test fixture requires it;
- adding sync/apply commands;
- validating Topic Catalog or Decisions prose tables as machine sources;
- adopting operating graphs for non-marketing teams;
- UI work;
- broad repository lint cleanup outside modified files.

## Target End State

After implementation:

- `checkable` and `contract` metadata graphs never disappear silently when malformed.
- `contract` validation enforces every relationship kind currently compared by diff, or explicitly documents and tests any intentionally diff-only relationship.
- `validate --team marketing-crew` and `diff --team marketing-crew` are both clean from the same reconciled baseline.
- Team-scoped graphs reject unknown local decisions unless the graph uses an explicit cross-team decision syntax accepted by docs and tests.
- `por:` validation either checks registered PoR authority or the docs are narrowed to say it is file-existence validation only. Choose one; do not leave code/docs implying different contracts.
- JSON responses use empty arrays for empty finding/diff collections.
- Topic Contract evidence entries render clearly when `for_decisions` is omitted.
- Tests cover the above behavior at parser, rule, API handler, CLI, and prompt-rendering seams.

## Implementation Strategy

### Phase 1: Preserve and Snapshot Baseline

Run and record the current clean state before edits:

```bash
prompt-manager graph operating-model validate --team marketing-crew --json
prompt-manager graph operating-model diff --team marketing-crew --json
prompt-manager graph topics --team marketing-crew --json
cd scenarios/prompt-manager/api && go test ./memberflow ./heartbeat
cd scenarios/prompt-manager/cli && go test ./graph
```

Do not proceed if marketing validate or diff is dirty. Reconcile design/config first.

### Phase 2: Make Graph Extraction Failure Semantics Explicit

Update `ExtractOperatingGraphBlocks` in `operating_graph_extract.go`.

Desired behavior:

- metadata block + malformed Mermaid fence returns an error for `mode: checkable` and `mode: contract`;
- `mode: explanatory` with graph metadata should also report parser errors unless there is a deliberate reason to support invalid explanatory diagrams;
- no metadata graph should be silently omitted after successful metadata parsing.

Recommended hard cut:

- remove the `if err != nil && meta.Mode == "contract"` conditional;
- return parse errors for all `prompt-manager-graph` blocks;
- update docs so "explanatory" means "no contract validation" but still parse-valid Mermaid subset when marked with `prompt-manager-graph`.

Tests:

- malformed `checkable` graph returns an error;
- malformed `explanatory` metadata graph returns an error if adopting strict parse;
- unmarked Mermaid elsewhere remains ignored by this parser.

### Phase 3: Promote Diff Relationship Kinds Into Completeness Rules

Add completeness rules one relationship family at a time in `operating_graph_rules_completeness.go`.

Recommended rule set:

| Runtime relationship | Existing diff kind | New validation rule | Severity |
|---|---|---|---|
| `topic_required_read` | `topic_read` | `graph_declared_required_read_missing` | error |
| `topic_evidence_consumed` | `topic_read` / `decision_consumed` | `graph_declared_evidence_missing` | error |
| `decision_consumed` | `decision_consumed` | `graph_declared_decision_consumed_missing` | error |
| `capability_gap_raised` | `capability_gap_raised` | `graph_declared_capability_gap_missing` | warning initially, then error after marketing is clean |
| `external_producer` | `external_producer` | `graph_declared_external_producer_missing` | warning initially |
| `cross_team_output` | `cross_team_output` | `graph_declared_cross_team_output_missing` | warning initially |

Implementation guidance:

- Reuse `OperatingGraphContractIndex`, `OperatingRelationshipSet`, and `OperatingRelationshipMatcher`.
- Do not add a second matcher path for validation.
- Keep each rule small and focused.
- For broad graph edges like `topic -> member`, use existing matcher semantics so read subtypes continue to map to the readable Mermaid shape.
- For evidence with `for_decisions`, ensure either `topic -> member` or `decision -> member` behavior is explicitly chosen and documented. Recommended: require `topic -> member` for the evidence source and allow `decision -> member` to satisfy decision-consumption visibility.

Tests:

- one unit test per new rule showing missing graph relationship fails;
- one unit test per rule showing the current marketing graph/config passes;
- one test proving validate and diff agree for all enforced relationship kinds.

### Phase 4: Define Validate/Diff Clean-State Semantics

Make the docs and CLI output precise.

Recommended contract:

- `validate` is the gate: contract graph is structurally valid and completeness-enforced for active rule stages.
- `diff` is the reconciliation map: any remaining diff is not necessarily a validation failure until its relationship family is staged into validation.
- once all current relationship families are staged, marketing should have both `validate` clean and `diff` clean.

Update:

- `docs/agent-system/OPERATING_GRAPHS.md`
- CLI help/next-step copy in `scenarios/prompt-manager/cli/graph/operating_model.go` if needed.

Tests:

- CLI validation text still prints clean state correctly;
- CLI diff text says no reconciliation required when diff is empty.

### Phase 5: Tighten Decision Scope

Update `graphUnknownDecisionRule`.

Recommended behavior:

- for `scope: team`, `decision:<id>` must resolve in that graph team's `team.json::operatingContract.decisionContexts`;
- if cross-team decision references are needed later, introduce explicit syntax such as `decision[team:monetization]:pricing-update` or a separate `external-decision:` kind in a future plan;
- do not silently accept a decision because any team declares it.

Tests:

- local team decision passes;
- decision declared only by another team fails;
- `capability-gap` follows the same local decision context rule unless intentionally globalized and documented.

Docs:

- update `OPERATING_GRAPHS.md` Typed Nodes and Edge Semantics to say decisions are team-scoped.

### Phase 6: Resolve PoR Validation Semantics

Choose one direction before editing:

Option A, recommended for this pass: narrow semantics.

- Keep code validating file existence only.
- Update docs to say `por:` currently checks path existence and edge backing via `topics.json` `por_file` outputs.
- Avoid claiming PoR registry validation until a separate PoR authority registry exists.

Option B: enforce registered PoR authority.

- Locate the authoritative document registry used by operating policy/storage map generation.
- Add a resolver seam in `OperatingGraphRuntime`.
- Validate `por:` against that registry and file existence.
- This is broader and should be its own follow-up if the registry is not already cleanly reusable.

For this implementation, use Option A unless the existing registry seam is already trivial to call from `memberflow`.

Tests:

- existing path passes;
- missing path fails;
- docs avoid saying stronger registration is enforced.

### Phase 7: Polish JSON Response Shapes

Ensure empty slices serialize as `[]`, not `null`.

Targets:

- `OperatingGraphValidationResult.Findings`
- `OperatingGraphDiffResponse.Diff`
- any list response affected by zero-value nil slices in the operating graph endpoints.

Implementation options:

- initialize slices in constructors/runners before returning;
- or add response normalization in handlers.

Recommended: initialize in domain functions (`ValidateOperatingGraphs`, `DiffOperatingGraphs`) so API and direct tests see the same shape.

Tests:

- handler validate clean response contains `"findings":[]`;
- handler diff clean response contains `"diff":[]`;
- CLI JSON output remains valid.

### Phase 8: Harden Topic Contract Rendering

Update `RenderTopicContract`.

Cases to cover:

- `EvidenceConsumedEntry{Prefix: "foo/*", ForDecisions: nil}` renders as general evidence, not `for ```.
- optional `comment` fields, if present in schema, are either rendered or intentionally ignored with tests/docs.
- outputs with `DestinationPORFile` render path clearly.

Recommended wording:

```markdown
- `foo/*` - general evidence
```

Tests:

- empty `for_decisions`;
- sorted decision-specific evidence;
- POR output path rendering;
- prompt section ordering remains `Operating Policy` -> `Topic Contract` -> `Inbox Flow`.

### Phase 9: Documentation Updates

Update `docs/agent-system/OPERATING_GRAPHS.md`:

- parse requirements for all marked graph modes;
- team-scoped decision semantics;
- validation rule table with new completeness rules;
- validate vs diff clean-state meaning;
- PoR current semantics;
- Topic Catalog / Decisions tables remain prose reference material, not validated inputs.

Update `docs/marketing/OPERATING_MODEL.md` only if needed:

- ensure the full graph still renders readably;
- keep the Topic Catalog and Decisions sections aligned with the reconciled graph;
- avoid adding claims that tables are machine-validated.

### Phase 10: Final Cleanup and Verification

Run focused formatting and tests:

```bash
gofmt -w scenarios/prompt-manager/api/memberflow/operating_graph*.go \
  scenarios/prompt-manager/api/heartbeat/topic_contract.go \
  scenarios/prompt-manager/cli/graph/operating_model.go

cd scenarios/prompt-manager/api && go test ./memberflow ./heartbeat
cd scenarios/prompt-manager/cli && go test ./graph
```

Run full prompt-manager package checks:

```bash
cd scenarios/prompt-manager/api && go test ./...
cd scenarios/prompt-manager/cli && go test ./...
```

Run contract commands:

```bash
prompt-manager graph topics --team marketing-crew --json
prompt-manager graph operating-model validate --team marketing-crew --json
prompt-manager graph operating-model diff --team marketing-crew --json
prompt-manager team prompt-matrix marketing-crew --json
```

Restart and health-check the scenario through lifecycle tooling, not direct binaries:

```bash
cd scenarios/prompt-manager && make stop
cd scenarios/prompt-manager && make start
cd scenarios/prompt-manager && make logs
```

If a `make health` target exists, use it. Otherwise use the lifecycle-reported API URL/health endpoint from logs or `prompt-manager` CLI status commands.

## Contract Decisions

1. `prompt-manager-graph` metadata means the graph is part of the machine-readable operating graph layer. It must parse even in `explanatory` mode.
2. Contract completeness should converge with diff. Diff-only relationship families are temporary staging states, not the final ideal.
3. Decisions are team-scoped by default.
4. PoR validation is file existence plus edge backing unless a real PoR authority registry is explicitly integrated.
5. Prose tables near graphs are not validation inputs in this plan.
6. Empty JSON collections should be arrays.

## Testing Plan

Add or update tests in:

- `scenarios/prompt-manager/api/memberflow/operating_graph_test.go`
- `scenarios/prompt-manager/api/memberflow/handlers_test.go`
- `scenarios/prompt-manager/api/heartbeat/topic_contract_test.go`
- `scenarios/prompt-manager/api/heartbeat/prompt_builder_test.go`
- `scenarios/prompt-manager/cli/graph/operating_model_test.go`

Minimum test coverage:

- parser failure behavior for `checkable`, `contract`, and marked `explanatory`;
- rule registry includes every new rule in deterministic order;
- each completeness rule fails on a minimal missing relationship and passes on a matching graph relationship;
- marketing bundled graph validates clean and diffs clean;
- handler JSON uses empty arrays;
- Topic Contract renders general evidence correctly;
- prompt-matrix still includes Topic Contract for all active marketing members.

Avoid tests that only assert implementation details. Prefer behavior assertions against public functions and API/CLI surfaces.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Completeness rules over-constrain readable Mermaid diagrams. | Keep broad `topic -> member` read semantics and document subtype mapping. |
| Decision scoping breaks legitimate cross-team references. | Fail fast now; add explicit cross-team syntax later when a real use case appears. |
| Validation becomes noisy for future/transitional topics. | Preserve `topic[future]`, `topic[old]`, and `process/future` non-actionable semantics. |
| Docs claim more than code enforces. | Update docs in the same pass; include tests where behavior is contract-level. |
| Handler response changes break CLI tests. | Update API and CLI tests together; preserve command names and flags. |

## Non-Goals / Prohibited Patterns

- Do not add sync/apply mutation commands.
- Do not make Mermaid the runtime source of truth.
- Do not validate Topic Catalog or Decisions prose tables.
- Do not add compatibility behavior for the old monolithic validator files.
- Do not add broad Mermaid syntax support.
- Do not hide known dirty graph/runtime drift with allowlists.
- Do not weaken current marketing clean-state tests to pass.

## Definition of Done

- `prompt-manager graph operating-model validate --team marketing-crew` is clean.
- `prompt-manager graph operating-model diff --team marketing-crew` is clean.
- `prompt-manager graph topics --team marketing-crew --json` is clean.
- `go test ./...` passes in both `scenarios/prompt-manager/api` and `scenarios/prompt-manager/cli`.
- Marked malformed graphs fail predictably instead of being dropped.
- Validation completeness includes the staged relationship families listed above.
- Docs accurately describe exactly what is validated.
- Empty response collections serialize as `[]`.
- Generated Topic Contract handles general evidence entries cleanly.
- No compatibility shims, duplicate validator paths, or dead code remain.
