You are executing a **scenario improvement** task for the Ecosystem Manager.

## Boundaries & Focus
- Work strictly inside `{{PROJECT_PATH}}/scenarios/{{TARGET}}/` (and its `.vrooli/`, `docs/`, and `requirements/` folders). If you uncover problems elsewhere, record them in your notes instead of editing unrelated paths.
- Respect the generator’s scaffold. Follow the current steering focus, avoid regressions, and stay within the existing architecture rather than redesigning the scenario from scratch.

## Quick Validation Loop (repeat until green)
1. `vrooli scenario status {{TARGET}}`
   - Shows failing checks, missing files, stale ports, and remediation hints. Use it to prioritize work and verify progress before handoff.
2. `vrooli scenario completeness {{TARGET}}`
   - Shows objective quality score (0-100) with breakdown of quality, coverage, quantity, and UI metrics. Review recommendations to identify gaps (missing tests, template UI, low routing complexity, etc.). Re-run after changes to verify improvements. Target 80+ for production readiness.
3. `scenario-auditor audit {{TARGET}} --timeout 240`
   - Capture the JSON (or summary) and explain any remaining security/standards violations.
4. Scenario test runner (document the exact command).
   - Tests must pass after your change, or you must clearly explain why they remain failing. Phase scripts automatically sync requirement coverage when tests run.
5. `vrooli scenario ui-smoke {{TARGET}}`
   - Ensures the production UI bundle loads, the iframe bridge is ready, and Browserless captures artifacts (screenshot, console, network).

## Working with PRD & Requirements
- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/PRD.md` is read-only unless the task explicitly grants edit access. Fix code/tests so the PRD reflects reality; don’t toggle checkboxes manually.
- Operational targets (P0/P1/P2) gain ✅ status automatically when their linked requirements’ tests pass. Confirm each target you touch is mapped to requirement IDs in `requirements/index.json` under the numbered operational-target folders (e.g. `01-<first-target-name>`, `02-<second-target-name>`, …). Add/link requirements if needed and keep the filesystem mirror of the PRD targets intact (no duplicate compatibility shims).
- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/test/playbooks/` mirrors the same target layout under `capabilities/<target>/<surface>/` plus an additional `journeys/` folder to validate user flows. When adding/editing workflows, place them in the correct folder, tag selectors via `ui/src/consts/selectors.ts`, refresh the README if the structure changes, and never leave temporary copies in legacy directories.
- When you run tests, ensure the commands cover the `[REQ:ID]` tags for the target you’re improving. Document the commands in PRD/Test Commands and your summary so future agents can reproduce the evidence. Running the full suite triggers automatic requirement sync.

## Documentation Updates
- **docs/PROGRESS.md** – append a new row with date, author, % change, and short description (this is the on-disk progress log).
- **docs/PROBLEMS.md** – log unresolved issues or follow-up tasks under the existing sections.
- **docs/RESEARCH.md** – add any new references or learnings if you discover them while fixing the scenario.
- **README.md** – update run/test instructions or dependencies if your change alters how the scenario operates.
- **requirements/README.md** – keep module descriptions current when you add requirement files. If you reorganize modules, update the README and ensure the numbered folders still align with PRD targets without adding bridge directories.
- **test/playbooks/README.md** – document any new surfaces or journey folders you add and ensure the per-target structure matches the requirements. Re-run the registry script (`node scripts/scenarios/testing/playbooks/build-registry.mjs --scenario {{PROJECT_PATH}}/scenarios/{{TARGET}}`) after changes.
These files are the canonical documentation sources for future scenario improvement agents to read.

## Implementation & Stack Expectations
- Honor the existing architecture. If you add UI, default to React + TypeScript + Vite + shadcn/ui + lucide unless the scenario already uses another stack.
- Keep changes scoped to the current steering focus, prove results with requirements-linked tests, log the result in `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/PROGRESS.md`, and leave TODOs for remaining gaps.
- Preserve `[REQ:ID]` annotations in tests so requirement tracking stays accurate.

## Security & Standards Guardrails
- `scenario-auditor` output defines pass/fail. If violations remain, explain why and what follow-up is required.
- Health checks must include `timeout 5` and meaningful responses; ports must stay within the scenario’s allocation range.

## Failure Recovery Snapshot
- If you cannot finish a fix, log it (component, severity 1-5, attempts, blockers) in your summary and `docs/PROGRESS.md`. Future agents depend on this context.
- Never ignore a failing test—either resolve it or document why it cannot yet be fixed.

## Cross-Scenario Awareness
- Before changing APIs/CLI outputs, search the repo and review `scenario status` to see who consumes them. Prefer additive or versioned endpoints.
- For integration testing, resolve other scenarios’ ports dynamically via `vrooli scenario port <name> API_PORT`.

## Final Handoff (Required Format)
1. **Validation Evidence** – Re-run the Quick Validation Loop (status, completeness, auditor, tests, ui-smoke) and list the exact commands plus where logs/output are stored. Include the completeness score before/after your changes and highlight which metrics improved.
2. **Changes & Files** – Enumerate the operational targets/requirements you advanced and which files you touched (PRD notes, README, requirements modules, docs/PROGRESS.md, code/tests).
3. **Current Scenario Health** – State what now works, what still fails, and any regressions you observed.
4. **Next Steps / Risks** – Capture follow-up tasks, blockers, or recommendations so the next agent can pick up instantly.

Use this structure for your final response; ecosystem-manager relies on it to track progress and plan the next iteration.

## Task Context

**Task ID**: {{TASK_ID}}
**Title**: {{TITLE}}
**Type**: {{TYPE}}
**Operation**: {{OPERATION}}
**Priority**: {{PRIORITY}}
**Category**: {{CATEGORY}}
**Status**: {{STATUS}}
**Current Phase**: {{CURRENT_PHASE}}

### Notes
{{NOTES}}
