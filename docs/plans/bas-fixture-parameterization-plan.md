# BAS Fixture Parameterization Plan

## Why
Browser Automation Studio (BAS) integration tests use workflow JSON playbooks plus reusable fixtures (`test/playbooks/__subflows`). Every capability workflow calls the same hard-coded fixture `@fixture/open-demo-workflow`, which navigates to the seeded Demo project and clicks a specific workflow card. When that selector fails even once (slow seed, renamed workflow, UI drift) dozens of requirements fail simultaneously. The shared seeds (`test/playbooks/__seeds/apply.sh`) also recreate the exact same Demo project for every run, so workflows cannot isolate state.

Parameterizing fixtures turns them into reusable subroutines with inputs. Playbooks will call `@fixture/open-workflow(project="Demo Browser Automations", workflow="Demo Smoke Workflow")` instead of a literal fixture ID. That decouples requirements from a single demo configuration, lets seeds generate unique projects per run, and keeps lint/coverage accurate. The change also unlocks richer authoring ergonomics (templated selectors, requirement propagation) and shrinks the blast radius of flaky selectors.

## Goals
1. Add a declarative parameter contract to fixture metadata so authors can describe required arguments, types, defaults, and associated requirement IDs.
2. Extend the fixture resolver (Python) to parse `@fixture/<slug>(key=value, ...)`, validate arguments, and substitute them into nested nodes/selectors before workflows hit Browserless.
3. Support parameter templating for selectors/messages, store references (e.g., `@store/demoWorkflowId`), and requirement propagation so linting understands coverage.
4. Update seeds + representative workflows to supply parameters, paving the way for per-run isolation and future reuse beyond the canonical Demo data.

## Implementation Plan

### 1. Metadata & Syntax
- Add `parameters` and optional `requirements` arrays to fixture metadata (e.g., `test/playbooks/__subflows/open-demo-workflow.json`). Each parameter entry: `{ "name": "project", "type": "string", "required": true, "default": "Demo Browser Automations" }`.
- Document the call syntax in `test/playbooks/README.md` and `scripts/scenarios/testing/playbooks/README.md`: `"workflowId": "@fixture/open-workflow(project=\"Demo\", workflow=@store.latestWorkflowName)"`.
- Keep legacy `@fixture/foo` calls working by treating omitted arguments as “use defaults”.

### 2. Resolver Enhancements (`scripts/scenarios/testing/playbooks/resolve-workflow.py`)
1. **Parsing** – Detect `@fixture/<slug>(params)` strings, reuse the existing selector argument parser to read key/value pairs, and pass them to `resolve_definition`.
2. **Validation** – Load fixture metadata, ensure parameters match declared types/enums, apply defaults, and produce actionable errors when something is missing.
3. **Templating** – Walk the fixture definition and substitute `${param}` tokens in strings (labels, selectors, URLs). Run selector resolution after substitution so manifest lookups still work.
4. **Store references** – Accept values that start with `@store/`. The resolver records the reference verbatim so runtime evaluation works, but still enforces that the parameter type is `string` unless we later add richer types.
5. **Requirement propagation** – If fixture metadata declares `requirements`, append them to the parent workflow’s metadata (e.g., `metadata.requirementsResolved`). Structure lint can use that to stop warning about fixture files lacking requirement IDs.

### 3. Runner / Lint Integration
- Ensure both `_testing_playbooks__read_workflow_definition` (scripts/scenarios/testing/playbooks/workflow-runner.sh) and `testing::integration::resolve_workflow_definition` (scripts/scenarios/testing/shell/integration.sh`) rely on the Python resolver so there is only one source of truth for parameters.
- Update `scripts/scenarios/testing/playbooks/build-registry.mjs` to ingest fixture metadata parameters, enabling docs/registry consumers to see the fixture API.
- Enhance `testing::integration::lint_workflows_via_api` to surface resolver errors (invalid params, missing defaults) before calling the API.

### 4. Seeds + Playbooks
- Modify `test/playbooks/__seeds/apply.sh` to create unique project/workflow IDs per test run, writing them to a small JSON artifact (e.g., `test/playbooks/__fixtures/seed-state.json`). Add a helper subflow that loads those IDs into store variables.
- Convert one representative workflow (toolbar undo) to pass explicit parameters into the new fixture. After validation, migrate the rest in follow-up passes.

## Risks & Mitigations
- **Backwards compatibility:** Existing fixture calls must continue to work. Keep defaults that mirror today’s Demo names, and only require explicit parameters when a fixture truly depends on variable input.
- **Selector templating order:** Templating must run before selector manifest resolution so lint remains accurate. Restructure the resolver to substitute parameter values first, then invoke the existing selector handling.
- **Input sanitization:** Parameter values feed selectors/URLs. Restrict accepted characters, require quoting for strings, and reject unescaped `)` or `,` to prevent malformed CSS. Future work could add enum types for high-risk inputs.
- **Store references:** Since values like `@store/demoWorkflowId` resolve at runtime, validation can only ensure the parameter is declared as a string. Keep store references as opaque strings and document that callers must hydrate those store keys earlier in the workflow.
- **Incremental migration:** Turning on strict validation everywhere could break dozens of flows. Roll out the resolver changes first (with defaults), then gradually convert fixtures/playbooks. Once the migration is stable, tighten lint to require parameter metadata.

## Next Steps
1. Implement resolver + metadata changes (no playbook edits yet) and add docs/tests.
2. Update seeds + one workflow to exercise the feature end-to-end.
3. Convert remaining fixtures/workflows and adjust lint/structure expectations.
