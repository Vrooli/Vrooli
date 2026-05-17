# Prompt Manager Operating Graph Architecture Hardening Follow-Up Plan

## Purpose

Fully harden the operating-graph contract implementation now that the marketing operating model validates cleanly and the first architectural pass is in place.

This plan turns the current architecture review findings into an executable implementation sequence. The goal is not to change the marketing team's operating model. The goal is to make the operating-graph subsystem durable enough to use as the standard team-contract layer across Prompt Manager teams.

## Required Reading

Run these first:

```bash
prompt-manager skill read implementation-plan-authoring screaming-architecture-audit boundary-of-responsibility-enforcement decision-boundary-extraction seam-discovery-and-enforcement utils-unification refactor invariant-discovery-and-enforcement
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Then read these source and docs files:

```bash
docs/agent-system/OPERATING_GRAPHS.md
docs/marketing/operating/OPERATING_MODEL.md
docs/plans/prompt-manager-operating-graph-contract-implementation-plan.md
docs/plans/prompt-manager-operating-graph-validator-refactor-plan.md
docs/plans/prompt-manager-operating-graph-contract-hardening-plan.md
docs/plans/prompt-manager-operating-graph-relationship-registry-refactor-plan.md
scenarios/prompt-manager/api/memberflow/operating_graph_types.go
scenarios/prompt-manager/api/memberflow/operating_graph_extract.go
scenarios/prompt-manager/api/memberflow/operating_graph_mermaid.go
scenarios/prompt-manager/api/memberflow/operating_graph_docs.go
scenarios/prompt-manager/api/memberflow/operating_graph_actor_resolver.go
scenarios/prompt-manager/api/memberflow/operating_graph_runtime.go
scenarios/prompt-manager/api/memberflow/operating_graph_relationship.go
scenarios/prompt-manager/api/memberflow/operating_graph_relationship_registry.go
scenarios/prompt-manager/api/memberflow/operating_graph_contract_index.go
scenarios/prompt-manager/api/memberflow/operating_graph_rules.go
scenarios/prompt-manager/api/memberflow/operating_graph_rules_entities.go
scenarios/prompt-manager/api/memberflow/operating_graph_rules_edges.go
scenarios/prompt-manager/api/memberflow/operating_graph_rules_completeness.go
scenarios/prompt-manager/api/memberflow/operating_graph_rules_docs.go
scenarios/prompt-manager/api/memberflow/operating_graph_rules_prompt.go
scenarios/prompt-manager/api/memberflow/operating_graph_diff.go
scenarios/prompt-manager/api/memberflow/operating_graph_coverage.go
scenarios/prompt-manager/api/memberflow/operating_graph_service.go
scenarios/prompt-manager/api/memberflow/topic_contract_renderer.go
scenarios/prompt-manager/api/memberflow/handlers.go
scenarios/prompt-manager/api/heartbeat/prompt_builder.go
scenarios/prompt-manager/api/heartbeat/topic_contract.go
scenarios/prompt-manager/cli/graph/operating_model.go
scenarios/prompt-manager/cli/graph/operating_model_test.go
```

Baseline commands:

```bash
cd /home/matthalloran8/Vrooli
prompt-manager graph operating-model validate --team marketing-crew --json
prompt-manager graph operating-model diff --team marketing-crew --json
prompt-manager graph operating-model coverage --team marketing-crew --json
cd scenarios/prompt-manager/api && go test -count=1 ./memberflow ./heartbeat
cd ../cli && go test -count=1 ./graph
```

Observed baseline on 2026-05-07:

- `prompt-manager graph operating-model validate --team marketing-crew --json` returns `0 errors`, `0 warnings`.
- `prompt-manager graph operating-model diff --team marketing-crew --json` returns `diff_count=0`.
- `prompt-manager graph operating-model coverage --team marketing-crew --json` reports all relationship families matched; Topic Catalog rows `25/25`; Decisions rows `14/14`; topic-contract section/source coverage `6/6`; `topic_contract_content_parity=enforced`.
- `cd scenarios/prompt-manager/api && go test -count=1 ./memberflow ./heartbeat` passes.
- `cd scenarios/prompt-manager/cli && go test -count=1 ./graph` passes.

## Problem Statement

The current operating-graph implementation is conceptually strong and currently clean for marketing, but several architectural seams are still too weak for broad adoption.

Current findings:

1. Prompt-section validation can treat derived/offline topic-contract sections as if they prove real heartbeat prompt inclusion.
2. Docs tables are parsed per file and attached to every marked graph in the file, which is risky when one document eventually contains multiple marked operating graphs.
3. Actor alias parsing is partly hardcoded to marketing-specific labels and member IDs.
4. Relationship semantics are centralized enough for matching, but diff suggestions/details and some coverage behavior still live outside the relationship registry.
5. Missing docs-table rules exist in code but are not registered, leaving the intended table-presence contract ambiguous.
6. The CLI operating-model command file duplicates large API response DTOs and mixes fetch, command parsing, and presentation concerns in one large file.
7. The existing documentation describes the implemented contract, but it does not yet document the stronger provenance, docs-surface scoping, actor-alias, and registry-ownership invariants this follow-up should establish.

These are not marketing design failures. They are contract-infrastructure hardening issues.

## Scope

In scope:

- prompt-section provenance and live-vs-derived validation semantics;
- docs-surface scoping for Topic Catalog and Decisions tables;
- actor reference and group alias ownership;
- relationship registry ownership of descriptions, suggestions, acceptable runtime fields, diff participation, and coverage participation;
- cleanup of docs-table missing-rule policy;
- CLI file structure and duplicated DTO reduction where practical;
- API/CLI/docs/test updates required by the above;
- targeted tests for parser, docs tables, actor resolver, registry, validation, diff, coverage, prompt sections, handlers, and CLI output.

Out of scope:

- changing marketing team topic design, member responsibilities, or operating model graph shape;
- adding operating graphs to non-marketing teams;
- implementing automatic sync/apply commands;
- broad UI work;
- broad repo lint cleanup unrelated to touched files;
- changing public endpoint names or CLI command names unless separately approved.

## Current Technical Context

Current domain flow:

```text
docs/**/*.md
  -> LoadOperatingGraphBlocks / ExtractOperatingGraphBlocks
  -> ParseOperatingMermaid
  -> ExtractOperatingGraphDocs
  -> BuildOperatingGraphRuntime
  -> NewOperatingGraphContractContext
  -> ValidateOperatingGraphs / DiffOperatingGraphs / BuildOperatingGraphCoverage
  -> API handlers
  -> CLI presentation
```

Key current implementation facts:

- `operating_graph_extract.go` walks `docs/` and attaches `ExtractOperatingGraphDocs(lines)` to every graph block from the same file.
- `operating_graph_mermaid.go` supports invisible `%% @node ID kind:value` annotations, so rendered Mermaid labels can remain human-friendly.
- `operating_graph_runtime.go` builds `derivedTopicContractPromptSections` by default, then `OperatingGraphService.withPromptSections` can replace them with live structured heartbeat prompt sections when a provider is wired.
- `operating_graph_relationship_registry.go` is the current semantic registry for relationship shape, runtime kinds, matching, acceptable fields, coverage participation, and validation metadata.
- `operating_graph_diff.go` still owns relationship suggestions and prose detail through switch statements.
- `operating_graph_rules_docs.go` defines `graphTopicCatalogMissingRule` and `graphDecisionsTableMissingRule`, but `DefaultOperatingGraphRules` does not register them.
- `operating_graph_actor_resolver.go` hardcodes marketing aliases such as `advertisers`, `any marketing member`, and individual marketing member names.
- `topic_contract_renderer.go` is shared by `memberflow` validation and `heartbeat` prompt generation.
- `prompt_builder.go` inserts `# Topic Contract` after `# Operating Policy` and before `# Inbox Flow`.
- `cli/graph/operating_model.go` is over 500 lines and duplicates API response structs locally.

Current public surfaces to preserve:

- `GET /operating-graphs`
- `GET /operating-graphs/validate`
- `GET /operating-graphs/diff`
- `GET /operating-graphs/coverage`
- `prompt-manager graph operating-model list`
- `prompt-manager graph operating-model validate`
- `prompt-manager graph operating-model diff`
- `prompt-manager graph operating-model coverage`

## Target End State

The operating-graph subsystem should have explicit, enforced boundaries:

```text
Markdown operating graph documents
  -> graph block extractor
  -> Mermaid parser
  -> docs-surface parser/scoper
  -> actor reference resolver
  -> runtime loader
  -> prompt-section provider with provenance
  -> normalized contract context
  -> relationship registry and indexes
  -> rule registry
  -> validate / diff / coverage services
  -> API handlers
  -> thin CLI commands and presentation
```

Target invariants:

- A validation rule that claims real prompt coverage must use live prompt sections, not derived/offline placeholders.
- Derived prompt sections may be used for offline coverage only when reported as derived/unavailable, not as enforced runtime proof.
- Each marked graph block owns or references its own docs surfaces; Topic Catalog and Decisions rows are not accidentally shared across unrelated marked graphs.
- Actor aliases are team-aware config or graph metadata, not marketing constants inside generic graph code.
- Relationship registry entries own all relationship-level semantics needed by validation, diff, and coverage.
- Docs-table presence policy is explicit: either tables are required for contract graphs or the missing-rule code is removed.
- CLI remains a thin API wrapper with stable commands, smaller domain files, and less duplicated response-shape risk.

## Implementation Strategy

### Phase 0: Baseline and Guardrails

Run the baseline commands from Required Reading. Preserve their output in the implementation handoff.

Do not start refactoring if marketing validation or diff is dirty. If a new mismatch appears, stop and report the mismatch before editing marketing config or docs.

Use `apply_patch` for manual edits. Avoid direct scenario execution; use existing scenario lifecycle commands if the scenario must be restarted.

### Phase 1: Model Prompt-Section Provenance

Problem:

- `BuildOperatingGraphRuntime` currently seeds every member with a derived `topic-contract` prompt section.
- That makes offline validation convenient, but it blurs whether validation proved the actual heartbeat prompt includes the generated section.

Implementation:

- Extend `OperatingGraphPromptSection` with a provenance field, for example:
  - `SourceKind string json:"source_kind,omitempty"` with values `live` and `derived`; or
  - `Derived bool json:"derived,omitempty"` if a smaller API shape is preferred.
- Set derived sections from `derivedTopicContractPromptSections` to derived provenance.
- Set provider-backed sections in `heartbeatPromptSectionProvider.SectionsForMember` to live provenance.
- Update `topicContractPromptSection` or prompt rules so:
  - `graph_prompt_topic_contract_missing` is satisfied only by live sections when validating through API/provider-backed flows;
  - offline validation without a provider reports prompt content parity as `unavailable` or `derived`, not `enforced`.
- Decide exact behavior for CLI/API service:
  - Recommended: `OperatingGraphService.Validate` includes live prompt sections when provider exists; if provider is nil, prompt rules should either skip with coverage `unavailable` or emit a distinct warning that live prompt proof is unavailable.
  - Do not report live prompt parity as enforced from derived placeholders.

Tests:

- Unit test derived prompt sections do not satisfy live prompt-content parity.
- Handler test provider-backed empty sections produce prompt missing findings.
- Handler test provider-backed real sections produce `content_parity=enforced`.
- Coverage test differentiates derived/unavailable from live/enforced.

Docs:

- Update `docs/agent-system/OPERATING_GRAPHS.md` prompt coverage section to define `live`, `derived`, `unavailable`, and `enforced`.

### Phase 2: Scope Docs Tables to Graph Blocks

Problem:

- `ExtractOperatingGraphDocs(lines)` currently parses the whole file once and attaches the same `Topic Catalog` and `Decisions` tables to every marked graph block in that file.
- This is correct for the current marketing file only because it has one marked contract graph. It is not safe as a general plan-of-record pattern.

Implementation:

- Introduce an explicit docs-surface scoping model. Recommended options:
  - **Nearest-section scoping:** a marked graph owns tables found after its Mermaid fence until the next `<!-- prompt-manager-graph:` block or the next top-level `#` heading.
  - **Metadata pointer scoping:** graph metadata can specify `topic_catalog_heading` and `decisions_heading`, defaulting to nearest `## Topic Catalog` and `## Decisions`.
- Recommended first implementation: nearest-section scoping with default heading names. It keeps docs easy to author and avoids new metadata until necessary.
- Refactor extraction so docs parsing receives a bounded line range for each graph block rather than the entire file.
- Preserve current marketing behavior: the full marketing contract graph should still see the existing Topic Catalog and Decisions tables.
- Add source-range fields to `OperatingGraphDocs` only if useful for debugging; otherwise keep response shape stable.

Tests:

- File with two marked graphs and two Topic Catalog tables attaches each table to the correct graph.
- File with one marked graph and one table after it preserves current marketing-like behavior.
- Table before a marked graph is not accidentally attached unless the final chosen scoping rule explicitly allows it.

Docs:

- Update `OPERATING_GRAPHS.md` to explain where Topic Catalog and Decisions tables must live relative to a contract graph.

### Phase 3: Move Actor Aliases Out of Generic Code

Problem:

- `DefaultOperatingActorResolver` hardcodes marketing members and groups. That blocks reuse across other teams and hides team-specific semantics in generic operating graph code.

Implementation:

- Prefer typed actor references in docs:
  - `member:researcher`
  - `member:brand-manager`
  - `team:monetization`
  - `external:operator`
- Introduce team-aware actor groups through config or graph metadata. Recommended design:
  - Add optional `actor_groups` to graph metadata or to a team-level config surface.
  - For graph metadata, keep it small and explicit:
    - `actor_group.advertisers: member:oss-advertiser, member:subscription-advertiser`
    - `actor_group.marketing-members: team-members`
  - If metadata parsing becomes too awkward, use a small `docs/agent-system/OPERATING_GRAPHS.md`-documented table under the graph, parsed by the docs-surface layer.
- Hard rule: the generic resolver should not know marketing member IDs.
- Keep built-in generic aliases only when genuinely universal:
  - `operator` -> `external:operator`
  - `system` -> `external:system`
  - exact typed refs
- For existing marketing prose, either:
  - update table cells to typed refs, or
  - add explicit marketing actor group declarations next to the graph.
- Make unknown aliases fail with `graph_docs_unknown_actor`.

Tests:

- Resolver expands metadata/config-defined `advertisers` for marketing.
- Same alias is unknown for a team without that declaration.
- `any marketing member` no longer falls back to hardcoded marketing members unless explicitly declared.
- Typed refs continue to work without aliases.

Docs:

- Document the actor alias mechanism and recommend typed refs for new tables.

### Phase 4: Finish Registry-Owned Relationship Semantics

Problem:

- The relationship registry owns much of the semantic model, but relationship descriptions and suggestions still live in diff switches.
- Adding a relationship family should not require editing multiple switch statements in diff and coverage.

Implementation:

- Extend `OperatingRelationshipSpec` with fields or callbacks for:
  - display statement templates;
  - graph-to-runtime suggestions;
  - runtime-to-graph suggestions;
  - graph relationship location/detail text;
  - runtime relationship detail text;
  - whether runtime-only relationships are diffed;
  - whether graph-only relationships are validation errors or warnings.
- Keep callbacks simple and pure. Avoid over-generalizing beyond current relationship families.
- Update `operating_graph_diff.go` to ask the registry for:
  - acceptable runtime fields;
  - relationship statement;
  - suggestions;
  - detail text.
- Update coverage to honor `DiffIncluded` rather than assuming all coverage specs are diff relationships.
- Add registry validation that fails tests if a relationship participates in diff but lacks suggestion/detail metadata.

Tests:

- Registry validation catches missing diff suggestion metadata.
- Diff output for all existing relationship kinds is unchanged or intentionally updated with documented clearer wording.
- Coverage still reports all nine graph-level relationship families and no marketing drift.

Docs:

- Update `OPERATING_GRAPHS.md` "Adding Relationship Families" to list the new registry-owned metadata.

### Phase 5: Resolve Docs-Table Missing-Rule Policy

Problem:

- `graphTopicCatalogMissingRule` and `graphDecisionsTableMissingRule` exist but are not registered.
- Docs currently describe tables as optional contract surfaces when absent, but current marketing uses them as enforced surfaces when present.

Decision:

- Recommended: keep tables optional for `checkable` graphs but required for `contract` graphs after this follow-up.
- Rationale: contract graphs should be human-auditable; the tables are the compact reference explaining topic purpose and decision ownership.

Implementation:

- If accepting the recommendation, register missing-table rules for `mode: contract`.
- Update coverage so absent required tables report `missing` and validation emits errors.
- If choosing to keep tables optional, delete the unused missing-rule types and update docs to explicitly say absence is allowed.
- Do not leave unregistered dead rule code.

Tests:

- Contract graph without Topic Catalog fails if required.
- Contract graph without Decisions fails if required.
- Checkable graph can omit tables if that remains the intended mode behavior.

Docs:

- Update validation rules table with whichever policy is implemented.

### Phase 6: Split and Thin the CLI Operating-Model Code

Problem:

- `cli/graph/operating_model.go` mixes local DTOs, command parsing, raw HTTP handling, output formatting, and coverage rendering in one large file.
- The CLI is still a thin API wrapper behaviorally, but the file shape is becoming a maintenance risk.

Implementation:

- Split into focused files under `scenarios/prompt-manager/cli/graph/`:
  - `operating_model.go`: command registration and subcommand dispatch only.
  - `operating_model_types.go`: response DTOs, or replace with shared generated/client DTOs if an existing pattern supports it cleanly.
  - `operating_model_client.go`: API path/query/raw JSON helpers.
  - `operating_model_output.go`: human-readable validation/diff/coverage/list renderers.
- Keep command names, flags, JSON output, and human output stable unless improving wording tied to new semantics.
- Continue using `cliutil.ParseInterspersed`.
- Do not move business logic into CLI. Any new contract decision belongs in API/memberflow.

Tests:

- Existing CLI graph tests pass.
- Add/adjust tests for prompt provenance wording if human coverage output changes.
- JSON output remains raw API JSON and includes new provenance fields if API adds them.

### Phase 7: Documentation and Marketing Contract Revalidation

Update docs:

- `docs/agent-system/OPERATING_GRAPHS.md`
  - prompt-section provenance;
  - docs-table scoping;
  - actor alias/group ownership;
  - registry-owned relationship metadata;
  - docs-table required/optional policy;
  - coverage statuses.
- `docs/marketing/operating/OPERATING_MODEL.md`
  - only update if actor references or docs-table placement need to change for the new parser/scoping policy.

Re-run marketing validation after every phase that changes semantics:

```bash
prompt-manager graph operating-model validate --team marketing-crew --json
prompt-manager graph operating-model diff --team marketing-crew --json
prompt-manager graph operating-model coverage --team marketing-crew --json
```

If the new stricter checks expose a real marketing contract/config mismatch, stop and report it rather than silently reshaping the marketing model.

## Contract Decisions

Public API:

- Preserve endpoint names and request filters.
- Add fields only when needed; do not remove existing JSON fields.
- If adding prompt provenance fields, keep them additive.

CLI:

- Preserve command names and flags.
- Preserve `--json` raw response behavior.
- Human output may add provenance/status lines if it clarifies the new semantics.

Validation:

- Contract graph validation should only claim runtime prompt enforcement from live structured prompt sections.
- Docs tables should either be required for `contract` graphs or the missing-rule code should be deleted. Preferred decision: required for `contract`, optional for `checkable`.
- Actor aliases must be declared or typed; generic code should not infer team-specific groups from hardcoded member IDs.

Relationship registry:

- The registry is the source of truth for relationship-level semantics.
- Diff, validation, and coverage should not each maintain separate relationship switch tables.

## Testing Plan

Run targeted tests throughout:

```bash
cd scenarios/prompt-manager/api && go test -count=1 ./memberflow ./heartbeat
cd scenarios/prompt-manager/cli && go test -count=1 ./graph
```

Add focused tests in:

- `operating_graph_mermaid` / extraction tests for docs-surface scoping.
- `operating_graph_actor_resolver_test.go` for team-declared aliases and unknown aliases.
- `operating_graph_registry_test.go` for registry completeness metadata.
- `operating_graph_test.go` or split test files for validation/diff/coverage behavior.
- `handlers_test.go` for provider-backed live prompt section behavior.
- `heartbeat/topic_contract_test.go` for renderer stability if prompt content or source metadata changes.
- `cli/graph/operating_model_test.go` for any changed output or DTO handling.

Run live contract checks:

```bash
cd /home/matthalloran8/Vrooli
prompt-manager graph operating-model validate --team marketing-crew --json
prompt-manager graph operating-model diff --team marketing-crew --json
prompt-manager graph operating-model coverage --team marketing-crew --json
prompt-manager graph topics --team marketing-crew --json
```

Optional broader gates after targeted tests pass:

```bash
cd scenarios/prompt-manager && make test
```

If broad `make test` fails on unrelated pre-existing lint/docs issues, report the unrelated failures with exact failing gates and keep the targeted operating-graph gates as the acceptance baseline.

## Rollout/Validation Checklist

- [ ] Baseline marketing validate/diff/coverage captured before edits.
- [ ] Prompt-section provenance added and documented.
- [ ] Derived prompt sections no longer count as live prompt enforcement.
- [ ] Docs tables scoped per graph block.
- [ ] Marketing Topic Catalog and Decisions still attach to `marketing-operating-model`.
- [ ] Marketing-specific actor aliases removed from generic resolver or explicitly declared.
- [ ] Registry owns diff suggestions/details/acceptable fields for all relationship families.
- [ ] Dead or unregistered docs-table missing-rule policy resolved.
- [ ] CLI operating-model code split without behavior regression.
- [ ] `docs/agent-system/OPERATING_GRAPHS.md` updated.
- [ ] Targeted API and CLI tests pass.
- [ ] Marketing operating graph validate/diff/coverage remains clean or newly exposed design mismatches are reported.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Stricter prompt provenance makes offline CLI/API validation appear weaker. | Represent provenance honestly in coverage. Use live provider-backed validation in API/service flows where available. |
| Docs-table scoping accidentally detaches the marketing tables. | Add a marketing-like fixture before refactor, then preserve it through implementation. |
| Actor alias removal makes current marketing tables noisy. | Convert cells to typed refs or declare marketing actor groups explicitly in the same phase. |
| Registry callback design becomes over-engineered. | Add only the metadata needed by current relationship families; keep callbacks pure and small. |
| CLI split causes import cycles or duplicated tests. | Keep all files in the same `graph` package; move code mechanically first, then improve naming. |
| Required table policy blocks simple contract drafts. | Keep required tables only for `contract` mode; use `checkable` for partial/draft diagrams. |

## Non-Goals / Prohibited Patterns

- Do not implement sync/apply.
- Do not adopt this graph contract across other teams in this pass.
- Do not silently change the marketing operating model shape to make tests pass.
- Do not keep legacy/fallback relationship matchers.
- Do not add marketing-specific aliases to generic graph code.
- Do not let CLI implement contract logic that belongs in API/memberflow.
- Do not leave dead unregistered rule code after deciding docs-table policy.

## Definition of Done

This plan is complete when:

- Prompt coverage clearly distinguishes live prompt proof from derived/offline render metadata.
- Contract graph validation no longer treats derived topic-contract sections as real heartbeat prompt enforcement.
- Docs tables are scoped to the correct graph block and tested with multi-graph documents.
- Actor aliases are declared/team-aware or replaced with typed refs; generic code has no marketing member fallback list.
- Relationship registry owns all relationship-level metadata used by validation, diff, and coverage.
- Docs-table missing-rule policy is implemented and documented.
- CLI operating-model command code is split into focused files while preserving public command behavior.
- `docs/agent-system/OPERATING_GRAPHS.md` reflects the implemented semantics.
- These commands pass:

```bash
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/api && go test -count=1 ./memberflow ./heartbeat
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager/cli && go test -count=1 ./graph
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model validate --team marketing-crew --json
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model diff --team marketing-crew --json
cd /home/matthalloran8/Vrooli && prompt-manager graph operating-model coverage --team marketing-crew --json
```

