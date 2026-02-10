# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **scientific-debugging** — Hypothesis generation methodology.
- **visited-tracker-tools** — Track which code paths have been examined.

## Analysis Approaches
- Use `ast-grep` for structural code search across scenarios.
- Use `git log` and `git diff` to identify recent changes near the bug.
- Trace data flow from input to the point of failure.

## Usage Rules
- Never propose a hypothesis you cannot design a test for.
- Always consider timing, configuration, and cross-scenario interactions.
