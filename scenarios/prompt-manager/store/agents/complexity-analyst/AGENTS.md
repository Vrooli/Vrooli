# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Identify the target scenario for analysis.
- Review existing internal docs (PROBLEMS.md, SEAMS.md) if they exist.

## Workflow
1. **Scan codebase** — Read through scenario source files systematically.
2. **Identify hotspots** — Functions over 50 lines, nesting over 3 levels, duplicated blocks.
3. **Assess cognitive load** — How many concepts must be held in mind to understand each module?
4. **Check for utility sprawl** — Duplicated helpers, inconsistent naming, scattered concerns.
5. **Rank findings** — By impact (how much complexity removed) and effort (how hard to fix).
6. **Report to refactor-lead** — Structured findings with refactoring sketches.

## Skills
- `prompt-manager skill read cognitive-load-reduction` — Complexity reduction patterns.
- `prompt-manager skill read utils-unification` — Utility consolidation.
- `prompt-manager skill read concept-vocabulary-unification` — Naming consistency.
- `prompt-manager skill read domain-compression` — Abstraction compression.
- `prompt-manager skill read code-cleanup` — Code smell identification.

## Coordination
- Receive analysis scope from refactor-lead.
- Report findings with specific file paths and refactoring sketches.
- Work with refactor-engineer to validate that proposed simplifications preserve behavior.
