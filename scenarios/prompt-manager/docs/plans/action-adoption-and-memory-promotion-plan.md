# Action Adoption and Memory Promotion Plan

Status: ready for implementation.

## Purpose

Fully adopt prompt-manager Actions across agent memory promotion, meta-optimization operating policy, skills, prompts, documentation, seed Action rollout, and validation loops.

The Action entity itself is already implemented. This plan is the second-stage adoption plan: it teaches Vrooli when to use Actions, when not to use them, how notebook observations graduate into the right durable form, and how meta-optimization measures whether Actions are reducing token-heavy operational prose.

Canonical ontology:

```text
Truth lives in the Plan of Record.
Judgment lives in Skills.
Execution lives in Actions.
Implementation lives in CLIs.
Unbuilt work lives in the Backlog.
Raw learning starts in Notebooks.
```

## Required Reading

Run this before implementing any phase:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health team-shared-docs-design cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Read the Action and memory-promotion docs:

```bash
sed -n '1,260p' scenarios/prompt-manager/docs/concepts/ACTIONS.md
sed -n '1,220p' scenarios/prompt-manager/docs/concepts/MEMORY-PROMOTION.md
sed -n '1,260p' scenarios/prompt-manager/docs/plans/action-entity-and-memory-promotion-rfc.md
sed -n '1,260p' scenarios/prompt-manager/docs/plans/action-entity-implementation-plan.md
sed -n '1,220p' scenarios/prompt-manager/docs/concepts/HEARTBEATS.md
```

Read the current meta-optimization operating surfaces:

```bash
jq . scenarios/prompt-manager/store/teams/meta-optimization/team.json
sed -n '1,220p' scenarios/prompt-manager/store/teams/meta-optimization/shared/TEAM.md
sed -n '1,220p' scenarios/prompt-manager/store/teams/meta-optimization/members/skill-optimizer/HEARTBEAT.md
sed -n '1,180p' scenarios/prompt-manager/store/teams/meta-optimization/members/skill-optimizer/RESPONSIBILITIES.md
sed -n '1,220p' scenarios/prompt-manager/store/teams/meta-optimization/members/debt-curator/HEARTBEAT.md
sed -n '1,180p' scenarios/prompt-manager/store/teams/meta-optimization/members/debt-curator/RESPONSIBILITIES.md
sed -n '1,220p' scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/HEARTBEAT.md
sed -n '1,180p' scenarios/prompt-manager/store/teams/meta-optimization/members/run-introspector/RESPONSIBILITIES.md
sed -n '1,240p' scenarios/prompt-manager/store/teams/meta-optimization/members/meta-contrarian/HEARTBEAT.md
sed -n '1,180p' scenarios/prompt-manager/store/teams/meta-optimization/members/meta-contrarian/RESPONSIBILITIES.md
```

## Greenfield Constraint

This is greenfield adoption work. Do not include compatibility shims, legacy wrappers, deprecated aliases, dead code, unused re-exports, renamed `_unused` variables, or migration code for pre-Action adoption patterns.

Do not preserve old "skills convert directly into thin wrappers over scenario CLIs" language as a parallel policy. Replace it with the Action-aware policy:

```text
skill prose -> Vrooli-controlled CLI implementation -> Action contract -> skill collapse or retirement when appropriate
```

If a prompt, skill, or doc still says scenario CLI conversion is the final form, update it to make Actions the executable endpoint.

## Problem Statement

Prompt-manager now has implemented Action storage, API CRUD, validation, governed execution, CLI operations, AI indexing, opt-in mixed discovery, graph integration, UI browse/edit/run surfaces, and seed Actions. However, the operating layer still mostly behaves as if programmatic conversion ends at "thin wrapper skill over scenario CLI."

Current gaps:

- Meta-optimization `team.json` has no Action-specific decision context, promotion direction, knowledge topic, shared queue, or member lane ownership.
- `skill-optimizer` still evaluates conversion as skill-to-scenario-CLI instead of skill-to-Action when an Action contract can own the deterministic operation.
- `debt-curator` can promote notebook debt to skill, team-structure change, capability-gap, or retirement, but not Action.
- `run-introspector` captures repeated manual operations but has no explicit route for "this run should have used or created an Action."
- `meta-contrarian` has no failure modes for unsafe Action proposals, premature Action creation, missing CLI ownership, or Action-without-measurement.
- Agent tools files do not advertise `prompt-manager action` or mixed `prompt-manager discover --type all`.
- The Action graph already reports low inbound discoverability for seed Actions, proving adoption references are missing.
- `prompt-manager discover "list team decisions" --type all` currently returned only skills in local testing, even though `team.decisions.list` exists as a draft Action. Adoption must include search validation and fix-or-file behavior.
- The working notebook `docs/meta-optimization/CONVERSION_PLAYBOOK.md` still frames conversion as a scenario CLI wrapper, not an Action graduation pipeline.

The target is not to make every skill become an Action. The target is to keep each durable lesson in the cheapest reliable form that preserves its meaning.

## Scope

In scope:

- Meta-optimization team operating contract updates in `store/teams/meta-optimization/team.json`.
- Meta-optimization member prompt updates for skill-optimizer, debt-curator, run-introspector, toolchain-validator, and meta-contrarian.
- Meta-optimization shared artifacts and notebook docs.
- Relevant agent `TOOLS.md` updates for members that need Action discovery or Action proposal commands.
- Core skills that govern skill conversion, memory promotion, skill authoring, and plan discovery.
- Action adoption docs and docs manifest updates.
- Seed Action selection, validation, activation policy, and health measurement.
- Search/discovery validation for mixed Skill/Action results.
- Graph-health and usage measurement loops.
- Tests that protect the new policy from drifting back to CLI-wrapper-only language.

Out of scope:

- Rebuilding the Action entity implementation.
- Adding arbitrary shell or external-tool wrappers.
- Implementing missing scenario/resource/project CLI commands, except as backlog/capability-gap outputs.
- Converting every skill in one pass.
- Broad UI redesign beyond docs/prompt references needed for Action adoption.
- Cross-team rollout outside meta-optimization except for compact generic discovery guidance.

## Current Technical Context

Implemented Action surfaces:

- [CODE: store/schemas/action.schema.json] defines the Action contract.
- [CODE: api/actions/service.go] owns Action validation and governed execution.
- [CODE: api/actions/resolver.go] resolves Vrooli-controlled command ownership.
- [CODE: api/store/action_store.go] owns file-backed Action persistence.
- [CODE: cli/actions/actions.go] exposes `prompt-manager action` commands.
- [CODE: ui/src/components/action/ActionEditorPanel.tsx] exposes UI editing, validation, and run controls.
- [DOC: docs/concepts/ACTIONS.md] is the canonical Action concept.
- [DOC: docs/concepts/MEMORY-PROMOTION.md] is the canonical memory-promotion ontology.

Current seed Actions:

```bash
prompt-manager action list --json
```

As of this plan:

- `scenario.status.show` is active and runnable.
- `team.decisions.list` exists as a draft Action.

Meta-optimization contract surfaces:

- [CODE: api/teamcontract/contract.go] defines and renders structured team operating contracts.
- [CODE: api/teamcontract/contract_test.go] protects contract validity and prevents prompt prose from restating contract-owned policy.
- [CODE: store/teams/meta-optimization/team.json] is authoritative for decision contexts, caps, knowledge topics, notebook paths, write rules, and member task parameters.
- [CODE: api/heartbeat/prompt_builder.go] renders the operating contract into heartbeat prompts before member responsibilities.

Important existing constraint:

- The contract tests reject duplicated policy phrases in prompt prose. Keep caps, context ownership, document paths, and write rules in `team.json`; keep member markdown focused on judgment, workflow, and output shape.

## Target End State

At completion:

- Meta-optimization has an Action-aware operating contract.
- Notebook promotion has a first-class Action path.
- Skill optimization distinguishes:
  - irreducible judgment skill
  - skill section that should reference an existing Action
  - skill section that should graduate into a new Action
  - missing CLI implementation that belongs in backlog/capability-gap before an Action exists
- Repeated manual operations from run logs can produce Action candidates.
- Contrarian review rejects unsafe or premature Action proposals.
- Relevant agents discover executable operations with `prompt-manager discover --type all` and inspect/run Actions through `prompt-manager action`.
- The programmatic conversion queue tracks Action candidate/in-progress/completed lifecycle, not just skill-to-CLI wrapper lifecycle.
- Seed Actions have inbound graph references, usage expectations, and validation checks.
- Mixed Skill/Action discovery is tested and reliable enough for agent prompts to depend on it.
- Documentation and skills consistently use:

```text
If it says what is true -> Plan of Record.
If it says how to decide -> Skill.
If it says what to run -> Action.
If it says how it works -> CLI implementation.
If it says what is missing -> Backlog/capability-gap.
If it is unverified or one-off -> Notebook.
```

## Contract Decisions

### Action Is the Executable Endpoint

Do not continue treating "thin wrapper skill over CLI" as the final conversion state. The final executable state is an Action over one Vrooli-controlled CLI command.

Thin-wrapper skills remain valid only when they preserve judgment, safety boundaries, context, or workflow selection around Actions.

### Decision Contexts

Add Action-aware decision contexts to `meta-optimization`'s `operatingContract.decisionContexts`:

- `action-candidate`: propose a new Action or promote an existing draft Action to active.
- `action-improvement`: propose edits to an existing Action contract, examples, permission declarations, owner, validation, run eligibility, or discoverability references.
- `action-deprecation`: propose archiving an Action that is unused, unsafe, obsolete, or superseded.

Recommended ownership:

- `skill-optimizer`: `action-candidate`, `action-improvement`, `action-deprecation` when the trigger is a skill conversion or Action usage/discoverability issue.
- `debt-curator`: can propose Action promotion only through `meta-self-improvement` unless the contract intentionally gives it direct `action-candidate` ownership. Prefer keeping debt-curator proposal-only through `meta-self-improvement` to preserve lane discipline.
- `run-introspector`: should usually raise `run-lesson` that hands off to skill-optimizer or director-swarm. Give direct `action-candidate` ownership only if the team wants run-introspector to create Action proposals without skill-optimizer review. Initial recommendation: do not.
- `meta-contrarian`: reviews Action proposals through existing rejection/framework contexts.

### Knowledge Topics

Add Action-specific knowledge topics:

- `action-audit-YYYY-MM-DD`: skill-optimizer-owned snapshot of Action health/adoption review.
- `action-visited/<action-id>`: visited tracker for Action audit rotation.
- Optional later: `action-seed-rollout-YYYY-MM-DD` if seed rollout becomes large enough to need a separate snapshot.

### Shared Artifacts

Add these shared artifacts under `store/teams/meta-optimization/shared/`:

- `ACTION_AUDIT.md`: rolling audit of Action health, usage, graph inbound references, validation state, and adoption opportunities.
- `ACTION_CONVERSION_QUEUE.md`: candidate/in-progress/completed queue for skill/notebook/run-derived Action promotion.

Rename or replace the existing `PROGRAMMATIC_CONVERSION_QUEUE.md` only if doing so is a clean greenfield replacement in one phase. Do not maintain two overlapping queues long term. Preferred path:

1. Create `ACTION_CONVERSION_QUEUE.md`.
2. Move the single current adjacent note into it if still relevant.
3. Delete or retire `PROGRAMMATIC_CONVERSION_QUEUE.md` from the contract and docs.

If the executing agent judges that deleting the old file is too broad for the phase, mark it as retired in place and remove all contract references in the same phase.

### Promotion Directions

Update debt-curator task parameters from:

```json
["skill", "team-structure-change", "capability-gap", "retirement"]
```

to:

```json
["plan-of-record", "skill", "action", "cli-backlog", "capability-gap", "team-structure-change", "retirement"]
```

Use `cli-backlog` when a deterministic operation exists in prose but no Vrooli-controlled CLI command exists yet. Use `action` only when one CLI command already exists or a draft Action exists and needs promotion.

### Safety Boundary

Action proposals must include:

- exact target CLI command
- owner type and ID
- input/output contract summary
- permission declarations
- validation result or reason validation is blocked
- `runEligible` recommendation
- baseline for current token/manual-work cost
- expected delta and measurement plan

Reject proposals that wrap raw external tools, shell pipelines, command conditionals, or multi-command workflows. If branching is needed, the owning CLI must implement it before Action creation.

## Implementation Strategy

### Phase 1 - Baseline Audit and Evidence Capture

Deliverables:

- Capture current Action registry and health:

  ```bash
  prompt-manager action list --json
  prompt-manager graph health --type action --json
  prompt-manager discover "show scenario status" --type all --limit 10 --json
  prompt-manager discover "list team decisions" --type all --limit 10 --json
  ```

- Validate current seed Actions:

  ```bash
  prompt-manager action validate scenario.status.show --json
  prompt-manager action run scenario.status.show --input='{"scenario":"prompt-manager"}' --dry-run --json
  prompt-manager action validate team.decisions.list --json
  ```

- Record findings in the implementation handoff and in the new `ACTION_AUDIT.md` once Phase 2 creates it.
- If mixed discovery does not return relevant Actions, determine whether the issue is indexing, scoring, draft filtering, discover rendering, or query matching. Fix it if local to prompt-manager discovery; otherwise record a specific follow-up in the plan/handoff.

Acceptance:

- The executing agent knows which Action adoption gaps are real before changing prompts.
- Any current discover/search defects are either fixed or tracked as blockers with exact commands.

### Phase 2 - Meta-Optimization Operating Contract

Deliverables:

- Update [CODE: store/teams/meta-optimization/team.json]:
  - add Action decision contexts
  - add Action knowledge topics
  - add `ACTION_AUDIT.md` and `ACTION_CONVERSION_QUEUE.md` shared-state artifacts
  - update `skill-optimizer` lane, owned contexts, required topics, and allowed writes if needed
  - update `debt-curator.taskParameters.promotionDirections`
  - update `run-introspector` task parameters to flag repeated manual operations as Action opportunities
  - update `meta-contrarian` task parameters or safety rules with Action review checks
- Update [CODE: store/teams/meta-optimization/shared/TEAM.md] only for high-level Action-aware mission language; do not duplicate caps or context lists.
- Add or update contract tests if the schema/renderer needs to expose new task parameters clearly.

Implementation notes:

- Keep all caps, context ownership, source paths, write rules, and required knowledge topics in `team.json`.
- Do not restate contract-owned policy in member markdown. Existing tests in [CODE: api/teamcontract/contract_test.go] protect this.

Acceptance:

```bash
cd scenarios/prompt-manager/api && go test ./teamcontract ./store
```

The bundled meta-optimization contract validates and renders for every active member.

### Phase 3 - Shared Action Audit and Conversion Artifacts

Deliverables:

- Add [CODE: store/teams/meta-optimization/shared/ACTION_AUDIT.md] with sections:
  - current Action registry
  - health and validation state
  - graph/discoverability status
  - usage/read-run signals
  - seed candidates
  - revisit queue
- Add [CODE: store/teams/meta-optimization/shared/ACTION_CONVERSION_QUEUE.md] with sections:
  - candidates
  - blocked on CLI/backlog
  - in progress
  - completed
  - rejected/retired
  - measurement follow-up
- Retire or replace [CODE: store/teams/meta-optimization/shared/PROGRAMMATIC_CONVERSION_QUEUE.md] so only one conversion queue is authoritative.
- Update `docs/meta-optimization/CONVERSION_PLAYBOOK.md` to describe Action graduation:

  ```text
  prose skill section -> existing/needed CLI -> Action -> skill collapse/retirement
  ```

Acceptance:

- There is one authoritative conversion queue.
- The queue distinguishes Action-ready candidates from CLI-backlog candidates.
- Docs no longer imply the final state is a skill wrapping a scenario CLI.

### Phase 4 - Member Prompt Adoption

Deliverables:

- Update skill-optimizer:
  - `HEARTBEAT.md`: evaluate convert/prune/improve/action in the right order.
  - `RESPONSIBILITIES.md`: define when to create/update/deprecate Actions.
  - `TOOLS.md`: add `prompt-manager action list/show/validate/run`, `prompt-manager discover --type all`, and Action graph health.
- Update debt-curator:
  - `HEARTBEAT.md`: apply the memory-promotion classifier and include Action/CLI-backlog/plan-of-record paths.
  - `RESPONSIBILITIES.md`: cite Action as a permanent-structure promotion direction while preserving "propose only."
  - `TOOLS.md`: add Action discovery/inspection commands.
- Update run-introspector:
  - `HEARTBEAT.md`: when a run repeats manual deterministic operations, classify whether the lesson is existing Action usage, new Action candidate, or CLI-backlog/capability-gap.
  - `RESPONSIBILITIES.md`: require run lessons to name Action opportunities when execution evidence supports them.
  - `TOOLS.md`: add discover/action inspection commands for empirical shortcut detection.
- Update toolchain-validator:
  - Add Action signal only where relevant: if toolchain validation repeatedly uses deterministic CLI checks, it may raise capability-gap or hand off an Action candidate.
- Update meta-contrarian:
  - Add Action-specific review checks:
    - unsafe command boundary
    - missing CLI ownership
    - premature Action before CLI exists
    - Action without baseline/measurement
    - Action sprawl where an existing Action could be extended
    - direct implementation by proposal-only members

Acceptance:

```bash
cd scenarios/prompt-manager/api && go test ./teamcontract
```

Prompt prose does not restate contract-owned policy, and members have clear Action-specific judgment rules.

### Phase 5 - Core Skill Adoption

Deliverables:

Update only skills that materially govern this workflow. Candidate skills:

- `skill-principles`
- `skill-authoring-tools`
- `skill-improvement-suggestions`
- `capability-extraction`
- `conversation-friction-analysis`
- `plan-skill-discovery`
- `implementation-plan-authoring`
- `team-shared-docs-design`

Required changes:

- Add the memory-promotion ontology where relevant.
- Replace CLI-wrapper-only conversion language with Action-aware language.
- Make `prompt-manager discover --type all` the default when an agent needs to learn how to do deterministic operational work.
- Keep `prompt-manager discover` skill-only behavior where the task is explicitly skill selection, such as implementation plan required-reading discovery.
- Make `prompt-manager action show <id>` the inspect-first command for Action candidates.
- Make `prompt-manager action run <id> --dry-run` the validation-before-execution pattern when execution is appropriate.

Do not bulk rewrite every skill containing the word "Action"; many skills use "Actions" as a generic section label. Target only the workflow-governing skills.

Acceptance:

```bash
prompt-manager skill read skill-principles skill-authoring-tools skill-improvement-suggestions capability-extraction --output combined >/tmp/action-adoption-skill-check.md
rg "thin wrappers over scenario CLIs|final state.*scenario CLI|scenario CLI wrapper" scenarios/prompt-manager/store/skills/packs/core docs/meta-optimization scenarios/prompt-manager/store/teams/meta-optimization -S
```

Any remaining CLI-wrapper-only language is either removed or explicitly framed as pre-Action/CLI-backlog work.

### Phase 6 - Heartbeat Discovery Guidance

Deliverables:

- Review [CODE: api/heartbeat/prompt_builder.go] and [DOC: docs/concepts/HEARTBEATS.md].
- Add one compact rendered heartbeat guidance line only if it can be done without bloating every prompt:

  ```text
  Before manual deterministic operational work, run `prompt-manager discover "<what you need>" --type all`; prefer an exact Action over prose instructions, and use Skills when judgment is required.
  ```

- Prefer adding this as a small prompt-builder section or contract-rendered guidance only if tests prove it is concise and appears once.
- Do not add long Action documentation into every heartbeat.

Acceptance:

- Prompt output for a representative meta-optimization member includes compact Action discovery guidance exactly once.
- Prompt output remains readable and does not duplicate `ACTIONS.md`.
- Existing heartbeat prompt builder tests pass.

### Phase 7 - Seed Action Rollout

Deliverables:

- Promote `team.decisions.list` from draft to active only after validation and permission review.
- Seed a small, high-leverage set of core Actions. Suggested candidates:
  - `team.decisions.list`
  - `team.knowledge.list`
  - `action.registry.list`
  - `skill.graph.health`
  - `scenario.logs.show` or `scenario.logs.tail` if a controlled Vrooli command exists
  - `scenario.test.run` only if lifecycle/test command behavior and output contracts are stable
- For each seed:
  - validate contract
  - dry-run
  - run only if read-only and safe
  - add examples
  - add tags that match likely natural-language discovery queries
  - add graph inbound references from relevant skill/team/member docs

Implementation notes:

- Do not wrap missing commands. If a needed CLI is absent, add a `capability-gap` or backlog item instead.
- Keep seed count small. Quality and discoverability matter more than registry volume.

Acceptance:

```bash
prompt-manager action list --json
prompt-manager action validate <seed-id> --json
prompt-manager action run <seed-id> --input='<example-json>' --dry-run --json
prompt-manager graph health --type action --json
```

No seed Action has validation errors. Active seeds have examples, owners, permissions, and at least one inbound reference.

### Phase 8 - Discovery and Search Reliability

Deliverables:

- Ensure Action discovery works for obvious queries:

  ```bash
  prompt-manager discover "show scenario status" --type all --limit 10 --json
  prompt-manager discover "list team decisions" --type all --limit 10 --json
  prompt-manager discover "run a prompt-manager action" --type all --limit 10 --json
  ```

- If Actions do not appear:
  - inspect Action indexing/reindex status
  - run `prompt-manager search-reindex` if needed
  - check draft filtering
  - check Action tags/descriptions
  - add tests for expected mixed discover results
- Update CLI/help/docs if `prompt-manager discover --help` still hides `--type` options.

Acceptance:

- Mixed discovery returns relevant Action results for seed Action queries.
- Text fallback and AI-backed search both handle Actions, or documented fallback limitations are explicit.
- Tests cover at least one Action result in `--type all`.

### Phase 9 - Measurement Loop

Deliverables:

- Define measurable adoption signals in `ACTION_AUDIT.md`:
  - Action count by status
  - active runnable Action count
  - validation pass/fail count
  - Action graph inbound warning count
  - Action run count from `runs.jsonl`
  - skill sections collapsed to Action references
  - estimated token reduction from collapsed prose
  - repeated manual operation count from run-introspector
- Add skill-optimizer heartbeat output fields for Action audit:
  - Action picked, if any
  - disposition: create/promote/improve/deprecate/no-action
  - baseline and expected delta
  - queue updates
- Add a post-adoption check after 4 heartbeats:
  - Were seeded Actions discovered?
  - Were they run successfully?
  - Did any agent keep doing the same manual operation?
  - Did skill reads decrease for converted operations?

Acceptance:

- The system can answer "did Actions reduce token/manual-work cost?" with real signals, not narrative.
- Every Action candidate has a baseline and measurement plan before operator review.

### Phase 10 - Documentation Health and Reference Sync

Deliverables:

- Update [DOC: docs/concepts/ACTIONS.md] and [DOC: docs/concepts/MEMORY-PROMOTION.md] only where implementation/adoption status changed.
- Update [DOC: docs/concepts/HEARTBEATS.md] if prompt guidance changes.
- Update [DOC: docs/reference/cli-commands.md] if CLI examples or `discover --type` help text changes.
- Update [DOC: docs/manifest.json] if new docs are added.
- Add `[CODE: ...]` references for new docs where useful.

Acceptance:

```bash
rg "Actions \\(Proposed\\)|Status: proposed|thin wrappers over scenario CLIs|scenario CLI wrapper" scenarios/prompt-manager/docs scenarios/prompt-manager/store/teams/meta-optimization docs/meta-optimization -S
```

Remaining hits are intentional historical references or explicitly marked as old status in completed plans/RFCs.

### Phase 11 - End-to-End Team Validation

Deliverables:

- Build a representative heartbeat prompt for at least:
  - `skill-optimizer`
  - `debt-curator`
  - `run-introspector`
  - `meta-contrarian`
- Verify each prompt contains:
  - resolved operating contract
  - Action-aware judgment rules where appropriate
  - no duplicated caps/context lists outside the rendered contract
- Run targeted tests:

  ```bash
  cd scenarios/prompt-manager/api && go test ./teamcontract ./heartbeat ./actions ./aisearch ./graph ./store
  cd scenarios/prompt-manager/cli && go test ./actions ./discover ./parity
  cd scenarios/prompt-manager/ui && pnpm run type-check
  cd scenarios/prompt-manager/ui && pnpm test -- src/components/action/ActionEditorPanel.test.tsx src/components/action/ActionListPanel.test.tsx src/components/search/SearchResultsList.test.tsx
  ```

Acceptance:

- Meta-optimization prompts are Action-aware and contract-clean.
- Action registry, discovery, graph, and CLI tests pass.

### Final - Cleanup and Scenario Verification

Because this plan modifies the prompt-manager scenario, the final implementation phase must:

- Fix all lint, type, and unit test issues in modified files, including pre-existing ones in touched files.
- Run:

  ```bash
  cd scenarios/prompt-manager/api && go test ./...
  cd scenarios/prompt-manager/cli && go test ./...
  cd scenarios/prompt-manager/ui && pnpm run type-check
  cd scenarios/prompt-manager/ui && pnpm test
  cd scenarios/prompt-manager && make lint
  cd scenarios/prompt-manager && make test
  ```

- Restart through lifecycle:

  ```bash
  cd scenarios/prompt-manager && make restart
  cd scenarios/prompt-manager && make status
  ```

- Smoke-test seed Action behavior:

  ```bash
  prompt-manager action list
  prompt-manager action validate scenario.status.show
  prompt-manager action run scenario.status.show --input='{"scenario":"prompt-manager"}' --dry-run
  prompt-manager discover "show scenario status" --type all --limit 10
  ```

If `make test` fails in known broad scenario orchestration outside this adoption slice, record the artifact path, exact failing phase, and whether targeted Action/meta-optimization validations passed. Do not hide scenario-level failures.

## Testing Plan

Unit and integration tests:

- `api/teamcontract`: operating contract validates, renders new Action contexts/topics/artifacts, and prompt prose avoids contract-owned policy duplication.
- `api/heartbeat`: rendered heartbeat prompts include Action discovery guidance once when implemented.
- `api/actions`: seed Actions validate and dry-run.
- `api/aisearch`: mixed discover returns Action results for Action-like queries.
- `api/graph`: Action inbound references are detected from updated skills/prompts/docs.
- `cli/actions` and `cli/discover`: Action commands and mixed discovery remain stable.
- UI Action tests if seed/action docs changes affect discover rendering.

Manual validation:

- Use the Action UI to inspect, validate, and dry-run a seed Action.
- Confirm `ACTION_AUDIT.md` and `ACTION_CONVERSION_QUEUE.md` are understandable without chat context.
- Confirm at least one skill or member prompt links to each active seed Action.

## Rollout and Validation Checklist

- [ ] Baseline Action registry, graph health, and discover results captured.
- [ ] Meta-optimization `operatingContract` includes Action contexts/topics/artifacts.
- [ ] Contract tests pass.
- [ ] `ACTION_AUDIT.md` created.
- [ ] `ACTION_CONVERSION_QUEUE.md` created and old conversion queue retired or replaced.
- [ ] Conversion playbook updated to Action graduation.
- [ ] Skill-optimizer, debt-curator, run-introspector, toolchain-validator, and meta-contrarian prompts updated.
- [ ] Relevant `TOOLS.md` files include Action commands.
- [ ] Core skills updated, scoped to workflow-governing skills only.
- [ ] Mixed discovery returns Actions for seed queries.
- [ ] Seed Actions validated, dry-run, and graph-referenced.
- [ ] Measurement loop documented.
- [ ] Docs manifest updated.
- [ ] Targeted tests pass.
- [ ] Scenario restarted and health checked.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Action sprawl | Registry fills with tiny low-value wrappers | Require usage baseline, owner, validation, examples, and measurement plan. Contrarian reviews sprawl. |
| Unsafe execution | Actions wrap shell/external tools or destructive commands | Preserve Action command boundary; reject raw tools and shell syntax; require permissions and validation. |
| Prompt bloat | Every heartbeat gets long Action guidance | Add one compact discovery rule; link docs for detail. |
| Contract drift | Markdown prompts restate caps/contexts differently from `team.json` | Keep policy in operatingContract; rely on teamcontract tests. |
| Search disappointment | Agents are told to use Actions but discover cannot find them | Make mixed discovery validation an early phase and blocker for broad prompt rollout. |
| Premature Action creation | Action exists before CLI is stable | Route missing/awkward CLI to backlog/capability-gap first. |
| Measurement theater | Proposals claim token savings without proof | Require baseline and post-adoption measurement fields in every Action candidate. |
| Notebook source-of-truth blur | Notebook entries remain cited after promotion | Curator must retire or link obsolete notebook entries when permanent structure lands. |

## Non-Goals and Prohibited Patterns

Do not:

- Reimplement the Action entity.
- Wrap raw external tools directly.
- Encode branching, conditionals, or multi-step workflows in Action contracts.
- Preserve a parallel "skill thin wrapper over scenario CLI" policy as the final state.
- Bulk-edit every skill that says "Action" as a generic section label.
- Let debt-curator implement permanent structure directly.
- Add compatibility shims or migration code for old non-Action adoption patterns.
- Create Actions for judgment-heavy workflows.
- Promote unverified notebook observations.

## Definition of Done

This plan is complete when:

- Meta-optimization's structured operating contract has Action-aware contexts, topics, artifacts, and promotion directions.
- The member prompts and relevant agent tool files tell agents how to discover, propose, validate, and measure Actions without duplicating contract-owned policy.
- Core workflow-governing skills distinguish Plan of Record, Skill, Action, CLI, Backlog, and Notebook correctly.
- The old conversion queue/playbook no longer treats CLI-wrapper skills as the final conversion state.
- At least two high-value seed Actions are active, validated, discoverable, and graph-referenced, unless validation proves only one is currently safe.
- Mixed discovery reliably returns Action results for obvious seed queries.
- Action adoption has a measurable audit loop.
- Documentation is registered in the manifest and has no stale "Actions are proposed" language outside historical plan/RFC context.
- Targeted tests pass, prompt-manager is restarted through lifecycle, and scenario health is verified.
