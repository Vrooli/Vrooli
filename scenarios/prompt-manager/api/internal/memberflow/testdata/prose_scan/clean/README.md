# Fixture: clean

Baseline. Every prose-side topic reference in this fixture is satisfied by a
matching declaration in `topics.json` or `skill.json::writes_to[]`. The
golden is an empty array.

What this proves:
- Fully aligned member prose (RESPONSIBILITIES.md cites a topic the member
  declares as `output[].prefix`) does not fire.
- A writer-skill SKILL.md that cites a prefix listed in its own
  `skill.json::writes_to[]` does not fire.
- An agent identity template (SOUL.md) that cites a prefix declared by a
  team that binds the agent does not fire.
- A docs file under `docs/agent-system/` that cites a globally-declared
  prefix outside any code block does not fire (and a CLI invocation inside a
  fenced ```bash block is correctly excluded).

Regression watch: any change here that adds a finding implies the scanner
has gained a false-positive surface. Investigate before updating the golden.
