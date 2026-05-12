# Skill Authoring

Universal quality bars and authoring guidance that apply to every skill, regardless of category. Per-category specifics (`steer`, `search`, `tools`, `practice`, `meta`, `platform`) live in the matching `skill-authoring-<category>` skill.

This is canon. The `skill-authoring` skill cites this file and contains only category-spanning authoring guidance for Steer skills (the `{{TARGET}}` placeholder, etc.); the per-category authoring skills cite this file for the universal quality bars.

Cites `LAYERS.md`, `PRIMITIVES.md`, and `PROMOTION_LADDER.md`.

---

## The shared mental model problem

Skills exist to solve a fundamental problem: **multiple agents across different sessions need to conceptualize problems the same way.**

Without shared mental models, each agent (or even the same agent across sessions) makes different architectural decisions, leading to divergent implementations, duplication, and inconsistency. Generic advice like "keep code DRY" doesn't prevent this — agents need concrete patterns they can consistently apply.

**The goal of every skill:** create a mental model so clear that any agent, in any session, would make the same structural decisions when facing similar problems.

---

## Universal quality bars

Every skill must include:

- **Clear intent statement** at the top (1–2 sentences). Answers: "If I only read this sentence, what would I prioritize?"
- **Boundary definition** (what is in scope and out of scope).
- **Convergence patterns** (decision trees, tables, or diagrams) when choices must be consistent.
- **Output expectations** describing what can/can't/must change or how results must be formatted.
- **Human-first CLI consumption** by default: prefer direct CLI output and avoid parser pipelines (`--json`, `--raw`, `jq`) unless default output is too long or ambiguous for reliable execution.
- **Selector-first workflows** by default: prefer stable human-readable selectors over extracted opaque IDs when tools support both.

For skills with **operational CLI complexity** (multi-step workflows, mutable state, external dependencies, or non-trivial failure modes):

- Include a dedicated section named **`Troubleshooting & Edge Cases`**.
- Keep failure matrices, rare gotchas, diagnostics, and manual recovery guidance in that section instead of spreading them across core workflow text.
- Keep the main workflow readable and focused on standard execution.
- Treat repeated troubleshooting clarifications as a tooling signal: prefer promoting them to CLI output contracts or tool capabilities before adding more prose (per `PROMOTION_LADDER.md`).

For simple/stable skills with no meaningful long-tail behavior:

- The section may be omitted, but state this explicitly (e.g., `No known operational edge cases for standard usage.`).

---

## Convergence patterns

The most effective skills provide **visual patterns** that agents can reference consistently. When something can be described as a decision tree, diagram, or table, agents gravitate toward it across sessions.

### Decision trees

For multi-step decisions, provide a visual flow:

```
                    Is this used in 2+ places?
                              │
              ┌───────────────┴───────────────┐
            YES                               NO
              │                                │
              ▼                                ▼
     Extract to shared/              Is this conceptually
     immediately                     reusable?
                                           │
                               ┌───────────┴───────────┐
                             YES                       NO
                               │                        │
                               ▼                        ▼
                       Design for reuse          Keep local
```

### Decision tables

For classification decisions, use YES/NO tables:

| Question | If YES | If NO |
|----------|--------|-------|
| Is it used by 2+ components? | Zustand store | Continue... |
| Will it persist across navigation? | Zustand store | Continue... |
| Is it purely ephemeral UI? | Local useState | Zustand store |

### Architecture diagrams

For layered systems, show the flow:

```
Components (consume state, never define shared state)
    ↓
Hooks (React state management, UI logic)
    ↓
Controllers (orchestration, business logic)
    ↓
Services (API calls, validation)
```

Visual patterns are unambiguous. Prose like "consider whether the state is shared" leaves room for interpretation. A decision table with explicit criteria does not.

---

## Principles over prescriptions

While convergence patterns provide structure, skills should still guide *thinking*, not dictate *steps*.

**Good guidance:**
- "Prioritize stability-critical code paths"
- "Validate data at system boundaries"
- "Handle all states explicitly: loading, error, empty, success"

**Bad guidance:**
- "Edit src/components/Button.tsx on line 42"
- "Add exactly 5 tests per file"
- "Complete this in 30 minutes"

Teach *what to care about*, not *what to do*. Agents should be able to apply the principles to novel situations — but with enough concrete patterns that they arrive at consistent solutions.

---

## Clear intent statement

Every skill opens with a clear statement of purpose:

```markdown
## <Category> focus: <Skill Name>

<1–2 sentence summary of what this skill steers toward>
```

Categories: `Steer`, `Platform`, `Search`, `Tools`, `Practice`, `Meta` (per `PRIMITIVES.md`).

The summary should answer: "If I only read this sentence, what would I prioritize?"

Examples:
- "Prioritize hardening React UI components against runtime crashes in `path:scenarios/{{TARGET}}/`."
- "Focus on test coverage for critical user journeys in `path:scenarios/{{TARGET}}/` rather than superficial coverage metrics."

(Steer-specific: the `{{TARGET}}` placeholder is required and is the subject of `skill-authoring`'s Steer-only guidance.)

---

## Boundary definition

Every skill must explicitly define what's IN scope and OUT of scope. This prevents conflicts between skills and keeps agents focused.

**What's IN scope:**
- List specific areas the skill addresses
- Name the types of changes that are appropriate

**What's OUT of scope:**
- List what should NOT be changed
- Name adjacent concerns that belong to other skills

Session-level constraints (do not add features, preserve behavior, etc.) are handled by **scope skills** rather than inline constraint sections. Skills with `defaultScope` in their metadata automatically include the appropriate scope when loaded with `--with-scope`.

---

## Skill structure

Skills follow a consistent structure that makes them scannable and predictable:

1. **Focus statement** — what this skill steers toward
2. **Required reading** (when applicable) — `prompt-manager skill read <skill-id>` and `path:docs/...` citations
3. **Tooling prerequisites** — required setup (optional)
4. **Core principles** — numbered sections with convergence patterns where applicable
5. **Audit section** — assessment checklist for existing codebases (optional, see below)
6. **Memory management** — comment/documentation guidelines and integration with memory-related tools (e.g., `visited-tracker`, `knowledge-observatory`)
7. **Output expectations** — what can/must be changed
8. **Troubleshooting & Edge Cases** — for skills with operational CLI complexity (omit explicitly if none)

The exact number of principle sections varies; the structure should feel consistent.

### When to include an audit section

Some skills need to assess existing codebases at any maturity level before prescribing changes. For these, include an audit section with:

- **Concrete discovery commands** (grep, find, etc.) to identify current patterns
- **Red flag checklists** that indicate problems to address
- **A documentation template** for recording findings

Audit sections make skills applicable to brownfield projects, not just greenfield development.

---

## Anti-gaming measures

Skills should distinguish between "real improvements" and "superficial changes" to prevent metric-gaming:

- "Avoid superficial changes that rename variables or restructure code without materially improving crash resistance."
- "Do not pad test counts with trivial tests that don't validate real behavior."
- "Focus on tests that would catch actual bugs, not tests that inflate coverage numbers."

If a skill can be gamed by making shallow changes, add explicit guidance about what "real" progress looks like.

---

## Agent memory loop

Agent sessions are stateless — when a session ends, all context is lost. For skills that involve investigation, audit, or systematic work, agents must explicitly read and write documentation to maintain continuity across sessions.

```
┌─────────────────────────────────────────────┐
│           AGENT MEMORY LOOP                 │
├─────────────────────────────────────────────┤
│                                             │
│  START SESSION                              │
│       │                                     │
│       ▼                                     │
│  READ existing findings docs                │
│  (understand what prior agents discovered)  │
│       │                                     │
│       ▼                                     │
│  DO the skill's work                        │
│  (informed by prior findings)               │
│       │                                     │
│       ▼                                     │
│  WRITE updated findings                     │
│  (so next agent can continue)               │
│       │                                     │
│       ▼                                     │
│  END SESSION                                │
│                                             │
└─────────────────────────────────────────────┘
```

Without READ: agent duplicates or overwrites prior work.
Without WRITE: discoveries are lost when session ends.

Skills that involve investigation should specify:

1. **What to read** at session start.
2. **What to write** at session end.
3. **Document template** for consistent structure.

The specific documents depend on the skill's purpose. A skill might use an existing doc pattern (e.g., `SEAMS.md` for boundary discovery), define a new doc type specific to its needs, or use multiple documents for different types of findings.

**Conventional location:** `path:docs/internal/` for agent-produced findings (not user-facing docs).

### Relationship to `visited-tracker`

The memory loop has two complementary parts:

| Tool | Tracks | Purpose |
|------|--------|---------|
| `visited-tracker` | Which files have been analyzed | Prevents re-investigating the same files |
| Findings docs | What was discovered | Preserves knowledge for future sessions |

Use both together for skills involving systematic codebase work.

---

## Protective comments

When skills include configuration blocks (tsconfig, eslint, etc.), wrap them in protective comments that explain:

- Why the configuration exists
- What problems it prevents
- What NOT to do

```jsonc
{
  "compilerOptions": {
    // ╔════════════════════════════════════════════════════════════════╗
    // ║  SAFETY-CRITICAL RULES - DO NOT REMOVE OR WEAKEN              ║
    // ║  These rules prevent runtime crashes. If you encounter errors:║
    // ║  ✅ DO: Fix with optional chaining (?.) or null checks        ║
    // ║  ❌ DON'T: Remove the rules or use @ts-ignore                 ║
    // ╚════════════════════════════════════════════════════════════════╝
    "strict": true
  }
}
```

Future agents need to understand *why* rules exist before they can safely modify them.

---

## Referencing other skills

When a skill depends on another, reference it explicitly using the prompt-manager CLI pattern.

Required reading:

```
- `prompt-manager skill read <skill-id>`
```

Optional reading:

```
- `prompt-manager skill read <skill-id> <skill-id>`
```

Only require what is essential; keep optional lists short and relevant. PoR citations look like:

```
- `path:docs/agent-system/<file>.md`
```

---

## Registration and metadata

To publish a skill:

1. Create the skill directory in `path:scenarios/prompt-manager/store/skills/packs/core/<skill-id>/`.
2. Add the following files:
   - `SKILL.md` — skill content
   - `skill.json` — metadata with `id`, `name`, `description`, `modes`, `tags`
3. Run `prompt-manager skill sync` to pick up changes.
4. Verify via `prompt-manager skill show <id>`.

---

## Avoid skill sprawl

Before creating a new skill:

- Search for existing skills that already cover the concept.
- Prefer extending an existing skill when the guidance naturally fits.
- Only create a new skill when it introduces a distinct, reusable mental model.

---

## Skill system constraints

- Do **not** create skills for one-off tasks (use direct instructions instead).
- Do **not** duplicate guidance that belongs in `CLAUDE.md`, scenario-specific docs, or canon (`path:docs/agent-system/`, `path:docs/<domain>/`).
- Do **not** restate framework canon (the layer mantra, the promotion ladder, the 9-layer model, the inbox-router-drain pattern). Cite `path:docs/agent-system/<file>.md` instead. The `team-member-capability-architecture-audit` skill flags this as the "skillless canon residue" smell.
- Prefer **updating existing skills** when guidance can be naturally extended.
- Skills should be **transferable** across scenarios via the `{{TARGET}}` substitution pattern (Steer skills only; see `skill-authoring`).

---

## Skill architecture heuristics

Use these heuristics when creating or evolving any skill:

- **Optimize for entropy control, not content volume.** Most skill failures come from unmanaged clarification growth, not missing facts.
- **Standardize only high-leverage constraints.** Avoid rigid global templates; enforce only structures that materially improve execution consistency.
- **Keep the primary path clean.** Standard execution should be easy to scan and run without long-tail context switching.
- **Isolate long-tail operations.** For CLI-operational complexity, centralize rare failures and manual recovery in `Troubleshooting & Edge Cases`.
- **Treat repeated prose as a product signal.** If the same workaround appears repeatedly, promote it to CLI output contracts or tool capabilities (per `PROMOTION_LADDER.md`).
- **Prefer promotion + retirement loops.** When tooling improves, remove superseded prose to prevent one-way growth.
- **Prefer layered fixes.** Use skill text as interim guardrails when needed, but prioritize durable CLI/tool improvements for recurring friction.
- **Use trigger-based governance.** Apply heavier structure when operational complexity is present, not based on category labels alone.
- **Preserve dual usability.** Default human-readable flows should be directly actionable; machine-readable paths remain deterministic when needed.
- **Track complexity budget drift.** If gates/steps/long-tail prose increase, require explicit rationale and a retirement plan.

---

## Destination over direction: maturity ladders for audit-shaped skills

**Convergence thesis.** Two agents on two machines running the same skill against the same scenario should produce compatible outputs. Skills converge when they are *less prescriptive about the path* and *more prescriptive about the destination*. A skill that names a verifiable end state lets agents arrive there from any starting point; a skill that lists steps without naming the destination invites divergent interpretations every session.

### What counts as an "audit-shaped skill"

An **audit-shaped skill** examines an existing scenario and produces findings/recommendations against a defined target state. The ~30 steer skills (architecture, temporal flow, API, CLI, interop, storage, testing-seams, etc.) are the canonical cohort.

Non-examples: Tools, Search, and Practice skills, and greenfield-directive steers (skills that prescribe how to build something new from scratch rather than assess an existing artifact). These may borrow the pattern, but it is not mandatory for them.

### The four ingredients (mandatory for audit-shaped skills)

1. **Scope Boundaries up front** — anchored to `path:scenarios/{{TARGET}}/`. State explicitly what is in scope, what is out, and what hands off to sibling skills. The opening paragraphs must let the agent know whether to keep reading.
2. **Named Maturity Ladder (L0–L5)** — every level gated by a *verifiable artifact* (something an agent can confirm with `ls`, `grep`, or a CLI assertion), not by an adjective. Each level has two columns: **What exists** (the concrete artifact at this level) and **When to stop here** (the criterion that says "no further work justified now"). "Improved" / "better" / "more idiomatic" are not levels.
3. **Decision / Archetype Model** — a table the agent walks row-by-row to classify the artifact under audit. Agents pick a deterministic row instead of synthesizing prose; two agents reading the same scenario must land in the same row.
4. **Durable-doc Output Anchors** — findings land in `path:scenarios/{{TARGET}}/docs/ARCHITECTURE.md`, `SEAMS.md`, `PROBLEMS.md` (or the closest existing equivalents) via `knowledge-observatory-tools`. **Do NOT create standalone `*_AUDIT.md` reports.** Standalone audit files freeze a single session's view and rot; durable docs accumulate findings across runs and survive agent churn.

### Canonical exemplars

- `path:scenarios/prompt-manager/store/skills/packs/core/temporal-flow-audit/SKILL.md` §2 "Temporal Maturity Model" — the cleanest existing implementation of the pattern.
- `path:scenarios/prompt-manager/store/skills/packs/core/screaming-architecture-audit/SKILL.md` §2 "Architecture Maturity Model" — parallel implementation in a different audit domain.

Cite these when teaching the pattern; do not paraphrase the ladders elsewhere.

### Orthogonality with `PROMOTION_LADDER.md`

Two ladders compose; they do not compete.

- **Destination axis (this section):** WHAT the skill describes — is the end state a named, verifiable artifact?
- **Implementation axis (`path:docs/agent-system/PROMOTION_LADDER.md`):** HOW the skill is implemented — prose → CLI wrapper → Action → retired.

A destination-clear skill is a *precondition* for climbing the promotion ladder: `development-toolchain-validator` can only mechanize artifact checks that the skill has actually named. Without destination clarity there is nothing for a CLI to check against, so an unconfigured skill stays at the bottom of the implementation ladder regardless of how well-written its prose is.

### Programmatic reading

`path:scenarios/development-toolchain-validator/` P1-005 "Skill Maturity Score" is the mechanical reading of destination-over-direction. The score is weighted by: *has structural config + has CLI tool assertions + all assertions pass + no conflicts*. Skills that name verifiable artifacts can be configured and scored; skills that describe direction without a destination cannot, and they sit at the bottom of the score distribution until they encode their ladder.

### Applicability

- **Mandatory** for audit-shaped skills (the steer cohort listed above).
- **Optional** for Tools, Search, Practice, and greenfield-directive steer skills — borrow when useful, do not force.

---

## Output expectations

When authoring or updating a skill, you may:

- Add or update skill files in the packs directory.
- Create new skill directories for genuinely new mental models.

You must:

- Preserve principle-based guidance style.
- Keep skills transferable across scenarios.
- Include scope boundaries and output expectations.
- Use convergence patterns when decision consistency matters.
- Cite canon (`path:docs/agent-system/<file>.md`) instead of restating it.

You must NOT:

- Restate framework canon — cite it.
- Use time-based constraints ("do X in 30 minutes").
- Hardcode file paths without `{{TARGET}}` in steer skills.
- Use output quotas ("add 5 tests per file").
- Write tool-specific instructions that may not apply to all scenarios when the skill is meant to be transferable.
- Use prose-only guidance where a decision tree or table would be clearer.
