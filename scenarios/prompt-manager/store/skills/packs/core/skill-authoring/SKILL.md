---
name: "skill-authoring"
description: "Authoring guide by skill kind: steer ({{TARGET}} rule, opening template, maturity source), practice, tools, platform, search and meta. Universal quality bars live in docs/agent-system/SKILL_AUTHORING.md."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta","steer"]
  tags: ["skill","authoring"]
  icon: "fileedit"
  status: "active"
  revision: 46
  createdAt: "2026-01-23T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager"]
  origin:
    kind: "authored"
---
## Meta focus: Skill Authoring

Kind-specific authoring guidance. The universal quality bars (intent statement, boundaries, convergence patterns, troubleshooting section, anti-gaming, registration, the promotion ladder) are owned by `docs/agent-system/SKILL_AUTHORING.md`; cite them, never restate them (R7). A skill carries judgment a tool cannot hold, stays short, and names the friction it removes (R10).

Pick the kind from `modes[0]` and apply only its section.

### Steer (`modes[0] = "steer"`)

Steers how one scenario is built or improved.

- Always focus on one scenario through the `{{TARGET}}` placeholder: file paths (`scenarios/{{TARGET}}/ui/src/`), audit commands, documentation paths, `visited-tracker --location scenarios/{{TARGET}}`. Never hardcode a scenario name.
- Open with the template below. Its first sentence answers "if I only read this, what would I prioritize?"

```markdown
## Steer focus: <Skill Name>

Prioritize **<one-sentence outcome>** in `scenarios/{{TARGET}}/<area>/` with <posture / quality bar>.

Your goal is to <what changes>, rather than <what to avoid>.
```

- Audit-shaped steer skills follow the destination-over-direction pattern owned by `SKILL_AUTHORING.md` (§"Destination over direction: maturity sources for audit-shaped skills"); apply it and restate none of it.
- Declare `modes[0] = "steer"` in the `SKILL.md` frontmatter. Regenerate `skill.json` with `skill-sidecar-gen`; do not edit the sidecar.

### Practice (`modes[0] = "practice"`)

A repeatable methodology for a class of problems, applied to any codebase: no `{{TARGET}}`.

- Give the approach and its decision points. Use phases with entry and exit criteria only when they help.
- Make it falsifiable: at least one point where a hypothesis is rejected and the flow routes back, not only forward. Choose the convergence form that fits (feedback loop, parallel tracks that synthesize, triage branches, condition table, exit checklist).
- Retain evidence where the work already keeps it; no mandatory paperwork per application.
- End with an anti-pattern table (`Anti-pattern | Why it fails | Better`) of this method's characteristic mistakes.

### Tools (`modes[0] = "tools"`)

How to use one tool, resource or scenario safely: prerequisites, safe defaults, key commands, verification, guardrails (never bypass lifecycle tooling, never mutate shared resources unverified). Keep the long tail under one section named exactly `Troubleshooting & Edge Cases`. A repeated troubleshooting entry becomes a CLI or tool improvement, not more prose. Reference content that `--help` already carries is deleted, not summarized. Scenario usage skills (decision tree, rung-labelled leaves, learning spine, in-use settings table) are covered in `scenario-skill-setup`. The long-form tools guide is `skill-authoring-tools`.

### Platform, search, meta

- **Platform** (shared packages and contracts): compatibility first, additive by default, say what downstream consumers must run.
- **Search** (where to find information): name the source of truth, the query that reaches it, and when to stop searching.
- **Meta** (governing the skill system): state the decision the skill changes; do not add a rule a tool can enforce.

### Before you add a skill

Check for sprawl first: if an existing skill's job is nearly the same, extend or merge it. A new skill needs the friction it removes and the second occurrence that justifies it (R10). Judge the result with `skill-validation`.
