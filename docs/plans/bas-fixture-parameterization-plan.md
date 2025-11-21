# BAS Fixture Parameterization Plan

## Why
Browser Automation Studio (BAS) integration tests use workflow JSON playbooks plus reusable fixtures (`test/playbooks/__subflows`). Every capability workflow calls the same hard-coded fixture `@fixture/open-demo-workflow`, which navigates to the seeded Demo project and clicks a specific workflow card. When that selector fails even once (slow seed, renamed workflow, UI drift) dozens of requirements fail simultaneously. The shared seeds (`test/playbooks/__seeds/apply.sh`) also recreate the exact same Demo project for every run, so workflows cannot isolate state.

Parameterizing fixtures turns them into reusable subroutines with inputs. Playbooks will call `@fixture/open-workflow(project="Demo Browser Automations", workflow="Demo Smoke Workflow")` instead of a literal fixture ID. That decouples requirements from a single demo configuration, lets seeds generate unique projects per run, and keeps lint/coverage accurate. The change also unlocks richer authoring ergonomics (templated selectors, requirement propagation) and shrinks the blast radius of flaky selectors.

## Goals
1. Add a declarative parameter contract to fixture metadata so authors can describe required arguments, types, defaults, and associated requirement IDs.
2. Extend the fixture resolver (Python) to parse `@fixture/<slug>(key=value, ...)`, validate arguments, and substitute them into nested nodes/selectors before workflows hit Browserless.
3. Support parameter templating for selectors/messages (scoped to explicit fields), store references (e.g., `@store/demoWorkflowId`), and requirement propagation so linting understands coverage without duplicating metadata.
4. Update seeds + representative workflows to supply parameters, paving the way for per-run isolation and future reuse beyond the canonical Demo data.

## Implementation Plan

### 1. Metadata & Syntax
- Extend fixture metadata with:
  - `parameters`: array of `{ name, type, required?, default?, enumValues?, description? }`. `type` accepts `string | enum | number | boolean`. `enumValues` is mandatory when `type=enum`. `default` may reference `@store/foo` when `type=string`. Extend the docs (`test/playbooks/README.md`, `scripts/scenarios/testing/playbooks/README.md`) with a grammar table that explains quoting rules, allowed characters, escaping, and how `@store/...` tokens are parsed so authors do not guess at the syntax.
  - `requirements`: optional array of requirement IDs covered by the fixture. These IDs propagate to parent workflows so structure lint recognizes coverage.
- Document the call syntax in `test/playbooks/README.md` and `scripts/scenarios/testing/playbooks/README.md`: `"workflowId": "@fixture/open-workflow(project=\"Demo\", workflow=@store.latestWorkflowName)"`, noting that complex strings (containing `@`, `/`, whitespace, or punctuation) must be quoted while bare identifiers can stay unquoted. The parser update should also reject unescaped `,` or `)` tokens to surface malformed calls quickly.
- Keep legacy `@fixture/foo` calls working by treating omitted arguments as “use defaults”. Raise resolver errors only when required parameters are truly missing.

### 2. Resolver Enhancements (`scripts/scenarios/testing/playbooks/resolve-workflow.py`)
1. **Parsing** – Detect `@fixture/<slug>(params)` strings, reuse (and extend) the existing selector argument parser to read key/value pairs, and pass them (plus the slug) to `resolve_definition`. The parser must accept quoted strings containing `/`, `@`, spaces, and escaped characters, while bare identifiers stay limited to a strict regex.
2. **Validation** – Load fixture metadata, ensure parameters match declared `type` / `enumValues`, apply defaults, and produce actionable errors when something is missing. Strings accept either quoted literals or `@store/foo` references; numbers/booleans reject store references. Document the exact error text so CI users see self-serve feedback.
3. **Templating traversal** – Deep-walk the resolved fixture definition, replacing `${fixture.param}` tokens only inside explicitly whitelisted string fields (selectors, labels, URLs, message text, nested workflow-call `data`). After substitution, run the existing selector/manifest resolution so lint remains accurate.
4. **Requirement propagation** – If fixture metadata declares `requirements`, bubble them up into the parent workflow’s metadata (e.g., collect under `metadata.requirementsFromFixtures`). Teach structure lint, coverage reporters, requirements registry builder, and `scripts/requirements/report.js` to merge these IDs with the requirement IDs referenced from `requirements/*.json`, deduplicate conflicts, and warn when fixtures cite unknown requirement IDs.
5. **Error surface area** – Ensure resolver errors bubble through both `_testing_playbooks__read_workflow_definition` and `testing::integration::resolve_workflow_definition` so lint/test output shows the precise fixture + parameter causing trouble before making API calls.

### 3. Runner / Lint Integration
- Ensure both `_testing_playbooks__read_workflow_definition` (scripts/scenarios/testing/playbooks/workflow-runner.sh) and `testing::integration::resolve_workflow_definition` (scripts/scenarios/testing/shell/integration.sh`) rely on the Python resolver so there is only one source of truth for parameters.
- Update `scripts/scenarios/testing/playbooks/build-registry.mjs` to ingest fixture metadata parameters and propagated requirements, enabling docs/registry consumers (and requirement reporters) to understand fixture APIs and coverage. The registry should emit parameter docs (type, description, defaults) so downstream tooling/UX can auto-generate reference tables.
- Enhance `testing::integration::lint_workflows_via_api` to surface resolver errors (invalid params, missing defaults) before calling the API. Fail fast when fixture metadata is malformed.
- Update requirement reporting scripts (`scripts/requirements/report.js` and friends) to read `metadata.requirementsFromFixtures` when aggregating coverage so fixture-supplied IDs count.

### 4. Seeds + Playbooks
- Modify `test/playbooks/__seeds/apply.sh` to create unique project/workflow IDs per test run, writing them to a small JSON artifact under `test/artifacts/runtime/seed-state.json`. Define the lifecycle: seeds create the artifact atomically, workflows load it through a helper subflow (e.g., `__subflows/load-seed-state.json`) that pushes the IDs into stable store keys, and cleanup removes the artifact. Document the canonical store names so fixtures can reference `@store/seed.projectId`, `@store/seed.workflowName`, etc., without duplicating logic.
- Update docs to instruct authors to call `load-seed-state` before parameterized fixtures when they need seed-driven inputs.
- Convert one representative workflow (toolbar undo) to pass explicit parameters into the new fixture. After validation, migrate the rest in follow-up passes.

- **Backwards compatibility:** Existing fixture calls must continue to work. Keep defaults that mirror today’s Demo names, and only require explicit parameters when a fixture truly depends on variable input.
- **Selector templating order:** Templating must run before selector manifest resolution so lint remains accurate. Restructure the resolver to substitute `${fixture.param}` tokens first, leave unrelated `${...}` content untouched, then invoke the existing selector handling.
- **Input sanitization:** Parameter values feed selectors/URLs. Restrict accepted characters, require quoting for strings, and reject unescaped `)` or `,` to prevent malformed CSS. Future work could add enum types for high-risk inputs.
- **Store references:** Since values like `@store/demoWorkflowId` resolve at runtime, validation can only ensure the parameter is declared as a string. Keep store references as opaque strings and document that callers must hydrate those store keys earlier in the workflow.
- **Incremental migration & rollout controls:** Turning on strict validation everywhere could break dozens of flows. Introduce a resolver feature flag (e.g., `WORKFLOW_FIXTURE_PARAMS_STRICT`) so we can deploy metadata parsing in “warn” mode, monitor CI, then flip to strict enforcement once the repo is converted. Define phased rollout (Phase 0 warn-only, Phase 1 lint errors, Phase 2 execution enforcement) and assign ownership for flipping the flag only after telemetry looks clean.
- **Testing coverage:** Add resolver/parser unit tests, registry builder coverage, seed artifact lifecycle tests, and at least one end-to-end workflow test exercising parameterized fixtures so regressions are caught before shipping.

## Next Steps
1. Implement resolver + metadata changes (no playbook edits yet) and add docs/tests.
2. Update seeds + one workflow to exercise the feature end-to-end.
3. Convert remaining fixtures/workflows and adjust lint/structure expectations.
