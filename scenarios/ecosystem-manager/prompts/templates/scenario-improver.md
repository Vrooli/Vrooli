You are executing a **scenario improvement** task for the Ecosystem Manager.

---

## Boundaries & Focus
- Work strictly inside `{{PROJECT_PATH}}/scenarios/{{TARGET}}/` (including `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/`, `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/`, `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/`). Log any off-scenario issues in your summary instead of fixing them.
- Respect the generator’s scaffold: you enhance operational targets, fix regressions, and prove them with tests.

---

## Quick Validation Loop (run in this order)
1. `vrooli scenario status {{TARGET}}`
   - Shows failing checks, missing files, stale ports, and suggested fixes. Use it to prioritize work and to verify progress before handoff.
2. `scenario-auditor audit {{TARGET}} --timeout 240`
   - Capture the JSON (or summary) and explain any remaining security/standards violations.
3. Scenario test runner (list the exact command you executed).
   - The task is incomplete if tests fail after your change or you cannot justify why they remain failing.
Re-run the loop until the sections you touched are green.

---

- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/PRD.md` is read-only unless task notes explicitly grant edit permission. Fix code/tests so PRD statements become true.
- Operational targets (P0/P1/P2) flip to ✅ automatically via requirement tracking. If status looks wrong, update requirements/tests—not the checkbox.
- Append dated progress entries to `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/PROGRESS.md` instead of editing PRD: include date, author, % change, and a short summary of what moved.
- Flag inconsistencies (e.g., PRD promises a feature that still fails) in your final report; do **not** silently alter the doc.

---

- Requirements live in `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/` (or `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/index.json`). Follow the schema in `docs/testing/guides/requirement-tracking-quick-start.md`.
- When implementing an operational target, reuse or extend the existing requirement IDs. Create new IDs only when the PRD calls for a distinct outcome.
- Tag every validating test with `[REQ:ID]`. Multiple test commands can back a single requirement; all must pass for the requirement to reach `complete`.
- Document each command you ran in PRD/Test Commands and in your summary so future agents can reproduce the validation.
- After tests run, sync results (`node {{PROJECT_PATH}}/scripts/requirements/report.js --scenario {{TARGET}} --mode sync` or the scenario’s helper) so dashboards + checkboxes update automatically.

---

## Improvement Priorities
1. **Broken or failing checks** from `scenario status` and `scenario-auditor`.
2. **Unimplemented P0/P1 operational targets** (focus on completing or unblocking one target at a time).
3. **Experience polish / performance / docs** once the above are stable.
Always fix regressions you introduce before picking a new target.

---

## Implementation & Stack Expectations
- Honor the existing architecture. If you add UI pieces, prefer React + TypeScript + Vite + shadcn/ui + lucide unless the scenario already uses a different stack.
- Keep changes surgical: update one operational target, prove it with tests, log the result in `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/PROGRESS.md`, and leave clear TODOs for remaining work.
- When editing tests, keep `[REQ:ID]` tags intact so requirement tracking stays accurate.

---

## Security & Standards Guardrails
- `scenario-auditor` output defines pass/fail. If violations remain, explain why and what follow-up is required.
- Health checks must include `timeout 5` and meaningful bodies. Ports must stay inside the scenario’s allocation range.

---

## Failure Recovery Snapshot
- If you cannot finish a fix, log it (component, severity 1-5, attempts, blockers) in your summary and optionally `docs/PROGRESS.md`. Future agents depend on this context.
- Never ignore a failing test; either fix it or state why it cannot yet be resolved.

---

## Cross-Scenario Awareness
- Before changing APIs/CLI outputs, check downstream consumers (search the repo + inspect status output). Prefer additive or versioned endpoints.
- For integration testing, resolve ports dynamically via `vrooli scenario port <name> API_PORT` instead of hard-coding.

---

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
