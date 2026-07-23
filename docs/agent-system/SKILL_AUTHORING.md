# Skill Authoring

Universal quality bars and authoring guidance that apply to every skill, regardless of category. Per-category specifics (`steer`, `search`, `tools`, `practice`, `meta`, `platform`) live in the matching `skill-authoring-<category>` skill; the `contract` category's specifics live in this file's §"Contract skills: machine-invoked workflow prompts".

This is canon. Every per-category authoring guide cites this file for the universal quality bars. The Steer category's guide is the bare `skill-authoring` skill; it holds the Steer-specific rules (the `{{TARGET}}` placeholder, the opening template, the maturity-source requirement).

Cites `LAYERS.md`, `PRIMITIVES.md`, and `PROMOTION_LADDER.md`.

---

## The shared mental model problem

Skills exist to solve a fundamental problem: **multiple agents across different sessions need to conceptualize problems the same way.**

Without shared mental models, each agent (or even the same agent across sessions) makes different architectural decisions, leading to divergent implementations, duplication, and inconsistency. Generic advice like "keep code DRY" doesn't prevent this — agents need concrete patterns they can consistently apply.

**The goal of every skill:** create a mental model so clear that any agent, in any session, would make the same structural decisions when facing similar problems.

---

## Skills are conditioning signals

A skill is not documentation that happens to be read by a model — it is a conditioning signal that shapes the distribution of behavior an executor agent produces. A prompt token does not carry its dictionary meaning; it carries a pointer into the model's prior. This reframe is what *generates* the quality bars below; use it to derive new rules, not just to follow the listed ones. Evaluate any skill through four lenses:

- **Focality.** Does each rule point at something the model already knows deeply? A named industry standard, format, or pattern ("write requirements in EARS form", "ASD-STE100 procedural prose", "Given/When/Then") invokes thousands of training documents' worth of coherent behavior for one phrase — a focal point that independent agents converge on without coordinating. Hand-rolled rule lists are low-precision pointers the model must compose at inference time, and composition is where drift lives. Before hand-rolling a rule-set, ask: what named standard, pattern, or genre already encodes this? A name only works as a focal point if it is real and widely documented — verify the standard exists before citing it. The `writing-standards` skill holds the placement map for prose standards.
- **Interpretive entropy.** How many divergent-but-compliant readings does the text admit? Decision tables, one meaning per term, and controlled procedural language shrink the space; prose conditionals and decision-hiding words widen it. The test is behavioral, not aesthetic: two agents reading the same skill against the same situation must produce the same decision.
- **Verifiability.** Can compliance be checked mechanically — by lint, CLI assertion, or a named artifact — rather than by judgment? (This is the lens behind "Destination over direction" below.)
- **Attention economy.** Every token competes for the model's attention. Twenty weak rules dilute each other; one named standard plus three sharp exceptions does not. Coherent flow is not cosmetic — a skill that reads as one argument conditions behavior better than an equal-length pile of rules, because contradictory or orthogonal rules split the attention budget. Complexity growth is attention debt: when gates, steps, or long-tail prose accumulate, require explicit rationale and a retirement path (`PROMOTION_LADDER.md`).

The test for invoking any concept by name: **name-and-delete, never name-and-keep.** A named standard earns its place only when it lets you delete the hand-rolled rules it replaces. A concept name added alongside the rules it describes is decoration — it costs attention and conditions nothing.

### Conditioning defect patterns

The lenses generate a small defect vocabulary. This table is the single source of truth: meta skills that audit other skills (`skill-validation`, `skill-improvement-suggestions`) and audit topics cite these rows by ID instead of restating them.

| # | Defect pattern | Signal | Fix |
|---|---|---|---|
| C1 | Hand-rolled rule cluster (focality) | 5+ style/format rules that a real, widely documented named standard already encodes | Adopt the standard by name and delete the rules it replaces (name-and-delete). Verify the standard is real first; the prose-standard placement map lives in `writing-standards` |
| C2 | Name-and-keep decoration | A standard or concept cited alongside the hand-rolled rules it describes | Delete either the name or the rules |
| C3 | Attention-splitting rule pile | Many weak, orthogonal, or contradictory rules aimed at one behavior | Consolidate into one coherent argument or one decision table |
| C4 | High interpretive entropy | Two compliant readings of a key instruction produce materially different executions | The fix is a decision, not more prose; prefer decision tables over prose conditionals |
| C5 | Unverifiable rules | Compliance judged by adjectives ("clear", "appropriate") rather than lint, CLI assertion, or named artifact | Restate as a mechanically checkable condition, or apply §"Destination over direction" |

Routing rule for audits: a **confirmed C4** — the divergence probe in `skill-validation` produced two materially different compliant executions — is an executability defect and stays a validation finding. C1, C2, C3, and C5 are conditioning-cost defects; they belong to `skill-improvement-suggestions`.

---

## Universal quality bars

Every skill must include:

- **Clear intent statement** at the top (1–2 sentences). Answers: "If I only read this sentence, what would I prioritize?"
- **Boundary definition** (what is in scope and out of scope).
- **Convergence patterns** (decision trees, tables, or diagrams) when choices must be consistent.
- **Output expectations** describing what can/can't/must change or how results must be formatted.
- **Human-first CLI consumption** by default: prefer direct CLI output and avoid parser pipelines (`--json`, `--raw`, `jq`) unless default output is too long or ambiguous for reliable execution.
- **Selector-first workflows** by default: prefer stable human-readable selectors over extracted opaque IDs when tools support both.
- **STE-100 procedural prose.** Skills are procedural instructions read by agents with no shared context — write procedure and rule text in ASD-STE100 Simplified Technical English: one instruction per sentence, active voice, imperative mood, one meaning per term used consistently, concrete commands and paths instead of categories. Avoid decision-hiding words in rules — canonical list: "robust", "comprehensive", "appropriately", "properly", "seamless", "handle", "improve", "enhance", "leverage", "as needed" — name the specific behavior instead. This list is the single source of truth; skills cite it rather than copying it. Rationale and "why" passages stay in normal explanatory prose.

For skills with **operational CLI complexity** (multi-step workflows, mutable state, external dependencies, or non-trivial failure modes):

- Include a dedicated section named **`Troubleshooting & Edge Cases`**.
- Keep failure matrices, rare gotchas, diagnostics, and manual recovery guidance in that section instead of spreading them across core workflow text.
- Keep the main workflow readable and focused on standard execution.
- Treat repeated troubleshooting clarifications as a tooling signal: prefer promoting them to CLI output contracts or tool capabilities before adding more prose (per `PROMOTION_LADDER.md`).

For simple/stable skills with no meaningful long-tail behavior:

- The section may be omitted, but state this explicitly (e.g., `No known operational edge cases for standard usage.`).

---

## Contract skills: machine-invoked workflow prompts

A **contract skill** is the prompt contract of a declared Agent Manager workflow node. The workflow JSON references it by `promptRef`; reconcile resolves the skill body into the node's prompt template; the engine renders it with the node's bindings and sends the result as the run's entire prompt. The reader is a headless run: no human, no conversation history, no retry beyond the engine's bounded schema repair. Category: `contract` (`modes[0]`, per `PRIMITIVES.md`).

**Shape from schema, choice from skill.** The workflow node's `resultSpec.schema` is the single source of truth for output shape, and the engine renders that schema into the run prompt and into the schema-repair prompt. Therefore:

- Do not restate the schema — field lists, enum values, nesting, required-ness — in skill prose. Restated shape drifts, and the agent already sees the authoritative schema.
- Do own the **outcome decision rules**. A schema can say the outcome enum is `completed | proposed | needs_attention | abstained`; it cannot say when each value is correct. That choice is the contract skill's core content, and it must be a decision table (observable end state → outcome value), not prose.

**Required content.** This list is the complete structure for a contract skill; the generic skill-structure template and the `## <Category> focus:` header do not apply, because every line of the body is rendered into the run prompt and must earn its tokens there:

1. **Task statement** — 1–3 STE-100 sentences: what the run does to which subject.
2. **Outcome decision table** — one row per outcome value the schema admits, keyed to observable end states (commands run, artifacts produced, checks passed). Two agents that reach the same end state must select the same outcome value.
3. **Variable legend** — one line per template binding (`{{.snapshot}}`, `{{.entity}}`, …): what it contains and what is authoritative about it. These are Go-template placeholders rendered by Agent Manager at run time; `prompt-manager skill read` shows them literally.
4. **Authority boundary** — what the run must not mutate, in STE-100. State it positively where possible ("write only inside `<path>`").
5. **Method by citation** — when the run needs doctrine (how to review, how to author a plan), cite it: `prompt-manager skill read <skill-id>`. Do not inline doctrine another skill or doc owns.

**The conservative-branch default.** When an affirmative row's predicate is not proven by evidence in hand, the run must select the conservative outcome and name the unproven predicate in the result. This applies wherever the conservative outcome routes to review rather than acting; a workflow whose conservative branch itself acts (rollback, deletion) must state its own explicit calculus instead. Affirmative rows are sufficient-condition predicates that partial or uncertain end states also satisfy — without this default, two honest agents at the same end state diverge on temperament, and the drift is always optimistic. Sharpen row predicates where you can ("the exact command from the snapshot's validation context, not a subset"); this default covers the end states no authored predicate anticipates.

Exemplar: `swarm-manager-workflow-plan-author` (task statement, outcome decision rules, method delegated to `implementation-plan-authoring`).

**Exempt from:** the audit section, the agent memory loop, `Troubleshooting & Edge Cases`, `{{TARGET}}` transferability, and boundary-against-over-application prose — a contract skill cannot be over-applied, because the workflow node fixes its scope, inputs, and lifetime.

**Validation:** the divergence probe (`skill-validation` §3.3) is the primary gate and runs against the outcome decision table: if two compliant readings of the same end state select different outcome values, that is a confirmed C4. Structure checks written for interactive skills do not apply. Validation also cross-checks the referencing workflow declaration: every template variable in the skill has a matching node binding, and the skill restates no part of the node's `resultSpec` schema.

---

## Convergence patterns

Decision trees, tables, and diagrams are the working forms of the interpretive-entropy lens: a table with explicit criteria admits one reading where prose like "consider whether the state is shared" admits many, and agents gravitate toward visual patterns across sessions. Three forms:

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

---

## Principles over prescriptions

While convergence patterns provide structure, skills should still guide *thinking*, not dictate *steps*. Teach *what to care about*, not *what to do*: agents must be able to apply a skill to novel situations, yet meet enough concrete patterns that they arrive at consistent solutions. The canonical good/bad guidance examples live in `PRIMITIVES.md` §Skill — cite them; do not restate them.

---

## Clear intent statement

Every skill opens with a clear statement of purpose:

```markdown
## <Category> focus: <Skill Name>

<1–2 sentence summary of what this skill steers toward>
```

Categories: `Steer`, `Platform`, `Search`, `Tools`, `Practice`, `Meta`, `Contract` (per `PRIMITIVES.md`). Contract skills are exempt from this header format — see §"Contract skills: machine-invoked workflow prompts".

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

**Status semantics.** `status` is `active` or `draft` (`draft` marks work-in-progress; it is surfaced as a draft flag in list/show). There is no soft-retired state: retirement is deletion via `prompt-manager skill delete <id>` (per `PROMOTION_LADDER.md`), and git history is the archive. Do not leave superseded skills in the pack with a `retired` label — no surface consumes it, so they keep polluting list, discover, and search results.

### `targetDimensions` (steer skills)

A **steer skill** — one Swarm Manager's closed-loop controller may select to
improve a target — additionally declares `targetDimensions` in `skill.json`: the
improvement dimensions it closes. This is the declared half of the controller's
skill → dimension capability map; the controller buckets open `test-genie`
findings by dimension and selects the steer skill whose declared dimensions
cover the heaviest open cluster.

```json
{
  "id": "ux",
  "name": "UX Improvement",
  "modes": ["steer", "ux"],
  "targetDimensions": ["accessibility", "visual", "ui"]
}
```

The dimension vocabulary's single source of truth is
`packages/maturity-go/dimensions/dimensions.json` (consumed by Ecosystem
Manager and described in `scenarios/swarm-manager/docs/concepts/DIMENSIONS.md`).
Every value must be a member of that vocabulary; EM excludes and warns on
undeclared/out-of-vocabulary dimensions. Non-steer skills omit the field.

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

## Destination over direction: maturity sources for audit-shaped skills

**Convergence thesis.** Two agents on two machines running the same skill against the same scenario should produce compatible outputs. Skills converge when they are *less prescriptive about the path* and *more prescriptive about the destination*. A skill that names a verifiable end state lets agents arrive there from any starting point; a skill that lists steps without naming the destination invites divergent interpretations every session.

### What counts as an "audit-shaped skill"

An **audit-shaped skill** examines an existing scenario and produces findings/recommendations against a defined target state. The ~30 steer skills (architecture, temporal flow, API, CLI, interop, storage, testing-seams, etc.) are the canonical cohort.

Non-examples: Tools, Search, and Practice skills, and greenfield-directive steers (skills that prescribe how to build something new from scratch rather than assess an existing artifact). These may borrow the pattern, but it is not mandatory for them.

### The four ingredients (mandatory for audit-shaped skills)

1. **Scope Boundaries up front** — anchored to `path:scenarios/{{TARGET}}/`. State explicitly what is in scope, what is out, and what hands off to sibling skills. The opening paragraphs must let the agent know whether to keep reading.
2. **Named maturity source** — either a skill-owned maturity ladder or a provider-owned maturity report. If the skill owns the ladder, every level is gated by a *verifiable artifact* (something an agent can confirm with `ls`, `grep`, or a CLI assertion), not by an adjective; each level has **What exists** and **When to stop here** semantics. If a health provider owns the ladder through `.vrooli/maturity.json`, the skill must name the default human CLI command (for example, `proto-health validate scenario {{TARGET}}`) and treat that output as the single source of truth. **Do not duplicate or summarize provider-owned L0-L5 ladders in skill prose.**
3. **Decision / Archetype Model** — a table the agent walks row-by-row to classify the artifact under audit. Agents pick a deterministic row instead of synthesizing prose; two agents reading the same scenario must land in the same row.
4. **Durable-doc Output Anchors** — findings land in `path:scenarios/{{TARGET}}/docs/ARCHITECTURE.md`, `SEAMS.md`, `PROBLEMS.md` (or the closest existing equivalents) via `knowledge-observatory-tools`. **Do NOT create standalone `*_AUDIT.md` reports.** Standalone audit files freeze a single session's view and rot; durable docs accumulate findings across runs and survive agent churn.

### Canonical exemplars

- `path:scenarios/prompt-manager/store/skills/packs/core/temporal-flow-audit/SKILL.md` §2 "Temporal Maturity Model" — the cleanest existing implementation of the pattern.
- `path:scenarios/prompt-manager/store/skills/packs/core/screaming-architecture-audit/SKILL.md` §2 "Architecture Maturity Model" — parallel implementation in a different audit domain.

Cite these when teaching skill-owned ladders; do not paraphrase their ladders elsewhere. Provider-backed skills should cite the provider CLI instead.

### Orthogonality with `PROMOTION_LADDER.md`

Two ladders compose; they do not compete.

- **Destination axis (this section):** WHAT the skill describes — is the end state a named, verifiable artifact or a provider-owned maturity report?
- **Implementation axis (`path:docs/agent-system/PROMOTION_LADDER.md`):** HOW the skill is implemented — prose → CLI wrapper → Action → retired.

A destination-clear skill is a *precondition* for climbing the promotion ladder: `development-toolchain-validator` can only mechanize artifact checks that the skill has actually named. Without destination clarity there is nothing for a CLI to check against, so an unconfigured skill stays at the bottom of the implementation ladder regardless of how well-written its prose is. When a provider owns the maturity ladder, destination clarity means the skill routes agents to the provider output and focuses its own prose on remediation.

### Programmatic reading

`path:scenarios/development-toolchain-validator/` OT-P1-002 "Skill Maturity Score" is the planned mechanical reading of destination-over-direction: a per-skill score from validation-run trends (duration, token cost, unexpected-mutation verdicts). Skills that name verifiable artifacts or provider-owned maturity commands can be configured and scored; skills that describe direction without a destination cannot, and they sit at the bottom of the score distribution until they encode their destination source.

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
