# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Understand what refactoring changes are being made.
- Review the test suite for the affected code.

## Workflow
1. **Baseline** — Run full test suite before refactoring. Record results.
2. **Monitor each step** — After each refactoring step, run tests.
3. **Compare results** — Any new failures? Any changed behaviors?
4. **Design additional checks** — If tests are insufficient, propose new test cases.
5. **Verdict** — Pass (behavior preserved) or Fail (behavioral change detected).
6. **Report to refactor-lead** — Verification results with evidence.

## Skills
- `prompt-manager skill read test` — Testing strategy.
- `prompt-manager skill read e2e-testing` — End-to-end validation.

## Coordination
- Receive notification of refactoring steps from refactor-engineer.
- Run tests after each step and report results to refactor-lead.
- Flag any behavioral changes immediately.
