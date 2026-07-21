## Meta focus: Skill Authoring

Steer-specific authoring guidance. Universal authoring quality bars (intent statement, boundaries, convergence patterns, output expectations, troubleshooting section, anti-gaming, memory loop, registration) live in canon and are not restated here.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md` — universal quality bars and authoring principles
- `docs/agent-system/PRIMITIVES.md` — what skills are vs. agents / teams / Actions / CLIs

---

### 1. When to use this skill

Use when authoring or updating a **Steer** skill (`modes[0] = "steer"`) — i.e., a skill that steers how some `path:scenarios/<scenario>/` is built or improved. For other categories use `skill-authoring-{platform,search,tools,practice,meta}`.

---

### 2. Steer skills must target a specific scenario

**Steer skills are always focused on a single scenario via the `{{TARGET}}` placeholder.** This is a hard requirement for all steer-mode skills.

The `{{TARGET}}` placeholder gets substituted with the actual scenario name when the skill is applied. This ensures:

- File paths reference the correct scenario (e.g., `scenarios/{{TARGET}}/ui/`, `scenarios/{{TARGET}}/api/`).
- Audit commands search the right directories.
- Documentation is written to the scenario's docs folder.
- The agent stays focused on one scenario rather than making sweeping cross-repo changes.

**Where to use `{{TARGET}}`:**

- File paths: `scenarios/{{TARGET}}/ui/src/`
- Audit commands: `rg "pattern" scenarios/{{TARGET}}/`
- Documentation paths: `scenarios/{{TARGET}}/docs/internal/`
- `visited-tracker` commands: `--location scenarios/{{TARGET}}`

---

### 3. Steer-specific opening template

```markdown
## Steer focus: <Skill Name>

Prioritize **<one-sentence outcome>** in `scenarios/{{TARGET}}/<area>/` with <specific posture / quality bar>.

Your goal is to <what changes>, rather than <what to avoid>.
```

The first sentence answers: "If I only read this sentence, what would I prioritize?" (Per the universal intent-statement rule in `docs/agent-system/SKILL_AUTHORING.md`.)

---

### 4. Steer-specific output expectations

When authoring or updating a Steer skill, you may:

- Add or update Steer skill files in `path:scenarios/prompt-manager/store/skills/packs/core/<skill-id>/`.
- Create new Steer skill directories for genuinely new mental models (per the sprawl check in `docs/agent-system/SKILL_AUTHORING.md`).

You **must:**

- Include `{{TARGET}}` in file paths, audit commands, and documentation paths.
- Cite canon (`docs/agent-system/<file>.md`) instead of restating universal quality bars.
- Set `modes[0] = "steer"` in `skill.json`.

You **must NOT:**

- Hardcode scenario names in steer skills — always use `{{TARGET}}`.
- Restate the universal authoring quality bars (intent statement, boundaries, etc.) — those live in `docs/agent-system/SKILL_AUTHORING.md`.

---

### 5. Maturity-source requirement for audit-shaped steer skills

Audit-shaped steer skills (those that examine an existing scenario and produce findings against a defined target state — architecture, temporal flow, API, CLI, interop, storage, testing-seams, etc.) **must** follow the destination-over-direction pattern. `docs/agent-system/SKILL_AUTHORING.md` §"Destination over direction: maturity sources for audit-shaped skills" owns the four mandatory ingredients, the canonical exemplars, the provider-backed rules, and the greenfield exemption — apply that section directly; restate none of it in the authored skill.

---

### 6. Boundaries

In scope:
- Authoring guidance unique to Steer skills (the `{{TARGET}}` rule, the steer opening template).

Out of scope:
- Universal authoring quality bars (`docs/agent-system/SKILL_AUTHORING.md`).
- Per-category specifics for non-Steer categories (their own `skill-authoring-<category>` skills).
- The promotion ladder and the layering rule (`docs/agent-system/PROMOTION_LADDER.md`, `docs/agent-system/LAYERS.md`).

No known operational edge cases for standard usage.
