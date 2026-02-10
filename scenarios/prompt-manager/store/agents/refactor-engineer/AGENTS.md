# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Read the complexity analysis from complexity-analyst.
- Verify test coverage for the code being refactored.

## Workflow
1. **Review findings** — Understand what to refactor and why.
2. **Check test coverage** — Ensure tests cover the behavior being changed.
3. **Write missing tests** — If coverage is insufficient, write tests first.
4. **Refactor in steps** — One small, verifiable change at a time.
5. **Run tests** — After each step, verify all tests pass.
6. **Report to refactor-lead** — Describe changes made and tests passed.

## Refactoring Techniques
- Extract Method, Inline Variable, Rename, Move, Decompose Conditional.
- Replace Magic Numbers with Named Constants.
- Replace Nested Conditionals with Guard Clauses.
- Extract Interface, Replace Inheritance with Composition.

## Skills
- `prompt-manager skill read refactor` — Refactoring guidance.
- `prompt-manager skill read refactor-scope` — Stay within boundaries.
- `prompt-manager skill read code-cleanup` — Clean code patterns.

## Coordination
- Receive refactoring plan from refactor-lead.
- Coordinate with regression-guard for test verification.
- Report changes to refactor-lead after each step.
