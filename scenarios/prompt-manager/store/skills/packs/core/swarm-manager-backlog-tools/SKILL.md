# Backlog Tools

## Purpose

Canonical reference for the swarm-manager backlog item data model and interaction patterns. All skills that read or write backlog item artifacts should defer to this document for folder structure, artifact schemas, reading order, CLI commands, and mutation rules.

**Scope boundaries:**
- **In scope**: folder structure, artifact schemas, reading order, CLI commands, mutation rules, troubleshooting
- **Out of scope**: processing behavior, completion summaries, verification checklists (see `swarm-manager-processing-guidance`)

## Folder Structure

Every backlog item lives at `{{ITEM_FOLDER}}` and follows this layout.

> **Filesystem path:** Backlog items are plain directories stored at `scenarios/swarm-manager/{ideas|research|fix|execute|chore}/{item-name}/`. The `{{ITEM_FOLDER}}` variable resolves to this absolute path at runtime when invoked through the swarm-manager workflow. Agents that know the item kind and name can read/write files directly on the filesystem as an alternative to the CLI commands below.

```
item-folder/
├── spec.json              # Item metadata (kind, title, description, status)
├── plan.md                # Implementation plan (primary artifact for execution)
├── workshop/
│   ├── round-001.json     # First workshop round (decisions, info, readiness)
│   ├── round-002.json     # Second workshop round
│   └── ...                # Additional rounds as refinement continues
├── research/
│   └── summary.md         # Research findings (if deep research ran)
├── archive/
│   └── ...                # User-provided materials and superseded artifacts
└── [user files]           # Any additional context added by user
```

### Subfolder Reference

| Subfolder | Creator | Purpose |
|-----------|---------|---------|
| `workshop/` | workshop agent | Stores workshop round files with decisions, info items, and readiness scores |
| `research/` | research agent | Stores feasibility research and findings |
| `archive/` | user / system | User-provided materials (prior scenario artifacts, requirements, designs) and superseded artifacts. Agents should read but not modify. |
| root | user / system | `spec.json` metadata, `plan.md` implementation plan, user-uploaded context files |

## Artifact Schemas

### `spec.json`

```json
{
  "kind": "idea | research | fix | execute | chore",
  "name": "kebab-case-name",
  "title": "Human-readable title",
  "description": "Full description of the item",
  "priority": 1,
  "status": "backlog | researching | ready | queued | in_progress | completed | archived",
  "tags": ["tag1", "tag2"],
  "created": "ISO-8601",
  "updated": "ISO-8601"
}
```

### `plan.md`

Markdown implementation plan with 13 sections. This is the primary artifact that executing agents receive as context. Structure:

1. **Purpose** — What is being built and why
2. **Problem Statement** — Symptom, root cause, solution overview
3. **Scope** — In scope / out of scope
4. **Current Technical Context** — Key files, components, architecture
5. **Target End State** — What the system looks like after
6. **Implementation Strategy** — Phased steps with dependencies
7. **Contract Decisions** — API/CLI/data model behavior
8. **Testing Plan** — Test cases and verification
9. **Rollout / Validation Checklist** — Step-by-step verification
10. **Risks + Mitigations** — Risk table
11. **Non-goals / Prohibited Patterns** — Anti-patterns
12. **Definition of Done** — Objective completion criteria

Sections may be `<!-- TBD -->` until populated through workshop rounds.

### `workshop/round-NNN.json`

Zero-padded 3-digit round numbers (`round-001.json`, `round-002.json`, etc.).

```json
{
  "round": 1,
  "generated_at": "ISO-8601",
  "readiness": {
    "problem_clarity": 0,
    "scope_defined": 0,
    "approach_solid": 0,
    "testable": 0,
    "risk_awareness": 0
  },
  "items": [
    {
      "id": "d1",
      "type": "decision",
      "topic": "Authentication approach",
      "context": "Why this matters and what was found",
      "options": [
        {"key": "A", "label": "OAuth with Google", "rationale": "Lowest effort, covers 90% of users"},
        {"key": "B", "label": "JWT with custom auth", "rationale": "More control, offline support"},
        {"key": "C", "label": "Other", "rationale": "Provide your own approach"}
      ],
      "selected": null,
      "freeform": null,
      "notes": null
    },
    {
      "id": "i1",
      "type": "info",
      "text": "Important finding or observation"
    }
  ],
  "plan_updates": "Description of plan sections created/updated this round"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `round` | int | Round number (1-indexed) |
| `generated_at` | string | ISO-8601 timestamp |
| `readiness` | object | 5 dimension scores, each 0-3 |
| `items` | array | Workshop items (decisions, info) |
| `plan_updates` | string | What changed in plan.md this round |

**Workshop Item Types:**

| Type | Required Fields | User Action |
|------|----------------|-------------|
| `decision` | id, type, topic, context, options, selected | Select an option key (A, B, C...) or choose "Other" with freeform input |
| `info` | id, type, text | Read-only |

**Readiness Dimensions (all scored 0-3):**

| Dimension | 0 | 1 | 2 | 3 |
|-----------|---|---|---|---|
| `problem_clarity` | No information | Vague idea | Clear, some unknowns | Fully understood |
| `scope_defined` | No scope | Rough boundaries | Mostly defined | Crisp in/out scope |
| `approach_solid` | No approach | General direction | Concrete strategy | Detailed phased plan |
| `testable` | No test plan | Vague criteria | Test cases identified | Complete test plan |
| `risk_awareness` | Not considered | Some risks noted | Key risks with mitigations | Comprehensive risk matrix |

### `research/summary.md`

Markdown document with sections: Executive Summary, Feasibility Assessment, Dependency Analysis, Implementation Approaches, Risks & Mitigations, Next Steps. See the relevant research skill for the full template.

## Source Authority Hierarchy

The backlog folder represents a refinement pipeline. Each stage builds on the previous, producing progressively more refined and authoritative output.

### Refinement Levels

```
Most refined / highest authority
  ┌─────────────────────────────────────────────────────┐
  │  plan.md             The implementation plan.        │
  │                      Primary source of truth for     │
  │                      what to build/fix/research.     │
  ├─────────────────────────────────────────────────────┤
  │  workshop/           User answers and decisions      │
  │                      from workshop rounds. Direct    │
  │                      user intent.                    │
  ├─────────────────────────────────────────────────────┤
  │  research/           Advisory findings — informs     │
  │                      but does not override user      │
  │                      decisions.                      │
  ├─────────────────────────────────────────────────────┤
  │  spec.json           Original description and        │
  │                      metadata. Superseded by plan.md │
  │                      when it exists.                 │
  ├─────────────────────────────────────────────────────┤
  │  archive/            Raw materials from a prior or   │
  │                      existing scenario. Least        │
  │                      refined — source material only. │
  └─────────────────────────────────────────────────────┘
Least refined / lowest authority
```

**Key principle:** `plan.md` is the most up-to-date and authoritative source. It represents the fully synthesized output of all workshop rounds. When `plan.md` exists, treat it as the primary source of truth. When it doesn't, reconstruct the specification from lower-authority sources.

### How Refinement Accumulates

The workshop loop is iterative. Users run workshop rounds, answer questions, accept/reject proposals. Each round builds on all previous rounds:

- **First workshop round**: Reads spec, archive, research → produces initial plan.md + round-001.json
- **Subsequent rounds**: Reads plan.md + all prior rounds → identifies gaps → presents targeted decisions → updates plan.md
- **User responses between rounds**: Selected options and freeform input from prior rounds are incorporated into the next plan.md update

This means `plan.md` is never stale in the way `archive/` can be. Each workshop round incorporates everything that came before it.

### Reading Order

When processing a backlog item, read artifacts in this order:

1. `plan.md` — if it exists, start here (it's the most refined source of truth)
2. `spec.json` — item metadata; description is superseded by plan.md when it exists
3. `workshop/` — review rounds for resolved decisions (may contain new selections since last plan update)
4. `research/summary.md` — advisory feasibility findings
5. `archive/` — raw materials; only use for content not already captured in plan.md
6. User-uploaded files — additional context

### Decision Authority Rules

When sources conflict, apply this precedence (highest to lowest):

1. **Resolved decisions** (with a `selected` value) in workshop rounds are **definitive** — always implement the selected option
2. **Freeform responses** on "Other" selections represent direct user intent — implement as specified
3. **Unresolved decisions** (`selected: null`) are open unknowns — do not assume an answer
4. **`plan.md`** supersedes `spec.json` and `archive/` — it was synthesized with more context
5. **Research findings** are advisory — they inform but do not override user decisions
6. **`archive/`** is raw source material — use it only when `plan.md` doesn't cover the same content

## CLI Commands

### Read item metadata
```bash
swarm-manager backlog get --kind <kind> --name <name>
```

### List files in item folder
```bash
swarm-manager backlog files --kind <kind> --name <name>
```

### Read a specific file
```bash
swarm-manager backlog file-get --kind <kind> --name <name> --path <relative-path>
# Example: swarm-manager backlog file-get --kind idea --name my-feature --path plan.md
# Example: swarm-manager backlog file-get --kind idea --name my-feature --path workshop/round-001.json
```

### Upload a file
Use `--stdin` with a heredoc to avoid shell quoting issues (apostrophes in text break `--content '...'`):
```bash
swarm-manager backlog file-upload --kind <kind> --name <name> --path <relative-path> --stdin <<'EOF'
<content>
EOF
```

## Mutation Rules

| Artifact | Who may write | Who may read |
|----------|--------------|--------------|
| `spec.json` | system, user | all agents |
| `plan.md` | workshop agent, user | all agents |
| `workshop/*.json` | workshop agent (items), user (answers/decisions) | all agents |
| `research/summary.md` | research agent | all agents |
| `archive/*` | user, system (when archiving scenario artifacts) | all agents (read-only) |
| user files (root) | user | all agents |

## Troubleshooting & Edge Cases

### 404 on `file-get`
The artifact hasn't been created yet. This is normal — not every workflow step runs. Check which workflows have completed by listing files with `backlog files`.

### Empty `archive/` folder
No previous runs have been superseded. The archive folder is created on demand when an agent re-runs a workflow step.

### Unresolved decisions in workshop rounds
Decisions with a null `selected` field have not been resolved by the user. Do not assume selections — treat unresolved decisions as open unknowns and flag them if they are critical.

### Missing `plan.md`
No workshop has run yet. Fall back to `spec.json` description plus any research and `archive/` materials as the working specification. See the source authority hierarchy for full fallback rules.

### Conflicting information
Apply the source authority hierarchy: resolved decisions > plan.md > research > spec.json > archive. See the "Decision Authority Rules" section for details.
