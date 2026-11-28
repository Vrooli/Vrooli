---
name: "capability-extraction"
description: "Decide which methodologies embedded in agent files, or repeated patterns in a scenario, are worth extracting into a reusable skill: the extraction test, the already-covered check and the shrink verification."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["practice","meta-optimization","extraction","methodology","agents"]
  icon: "package"
  status: "active"
  revision: 2
  createdAt: "2026-02-11T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Practice focus: Capability Extraction

Find methodology embedded in agent files (AGENTS.md, TOOLS.md; SOUL.md is context only) or repeated across scenarios, and decide what deserves a skill of its own. Judgment goes to skills; deterministic steps continue toward a CLI and an Action (promotion ladder in `docs/agent-system/SKILL_AUTHORING.md`). Author the result with `skill-authoring` and judge it with `skill-validation`. A skill is justified by the friction it removes and a second occurrence, not by one agent's habit (R10).

### The extraction test

Score each candidate pattern, one point per yes. Three or more means extractable; fewer stays in the agent file with a one-line reason.

- Is it a repeatable process, not a one-time instruction?
- Would another agent benefit from the same method?
- Can it be stated without the source agent's identity?
- Does it have decision points, not just "do X"?
- Could parts of it become CLI capabilities?

Look for numbered workflow steps with delegation points, structured decision criteria, inline input-to-action tables, shared communication formats, and verification checklists buried in prose. Deterministic routing with no judgment is a CLI or Action candidate, not a skill. A pattern in two or more agents ranks above one in a single agent; wait for the second occurrence unless the method is plainly general. Rank by agents that benefit times promotion potential, minus extraction cost, and draw a line; record a revisit trigger for what falls below it.

### Already covered?

Before creating a skill: search `prompt-manager skill list` for the domain and read the overlapping skill. Partial overlap means extend that skill; none means a new one. Group by method, never one skill per source agent.

### For each extraction, record

The candidate, the source agents, create or extend (and the id), a one-to-two sentence focus, what stays in the agent file (identity, coordination, agent-specific judgment), and which lines of AGENTS.md and TOOLS.md become a skill reference. Extract leaf methods before pipelines that compose them.

### Shrink verification (after it is implemented)

- The agent file references the skill and is measurably shorter (record the line count before and after).
- The skill works standalone for an agent that never saw the source agent.
- No case the agent used to cover is lost.
- The skill passes `skill-validation`.

### Anti-patterns

| Anti-pattern | Better |
|---|---|
| Extracting everything, or the agent's identity | Leave identity and agent-specific judgment in the agent file |
| One skill per agent | Group by method |
| Skipping the already-covered check | Search first; duplicates diverge |
| Extracting deterministic logic | Flag it as a CLI or Action candidate |
| Extracting without verifying afterward | Run the shrink verification |
