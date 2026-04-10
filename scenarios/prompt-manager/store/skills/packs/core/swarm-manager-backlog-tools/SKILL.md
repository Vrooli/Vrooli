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
├── handoff/               # (idea only, generated at process-time) downstream execution package
├── workshop/
│   ├── round-001.json     # First workshop round (decisions, info, readiness)
│   ├── round-002.json     # Second workshop round
│   └── ...                # Additional rounds as refinement continues
├── research/
│   └── summary.md         # Research findings (if deep research ran)
├── archive/
│   └── ...                # User-provided materials and superseded artifacts
├── orchestration-summary.md  # (optional) Meta-orchestrator session summary
└── [user files]           # Any additional context added by user
```

### Subfolder Reference

| Subfolder | Creator | Purpose |
|-----------|---------|---------|
| `workshop/` | workshop agent | Stores workshop round files with decisions, info items, and readiness scores |
| `handoff/` | swarm-manager execution code | Idea-only execution package generated from the latest finalized backlog state; contains `brief.md`, `manifest.json`, and `source-index.json` for downstream ecosystem-manager tasks |
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
  "depends_on": ["kind/name", "fix/auth-bug"],
  "initiative": "initiative-name",
  "acceptance_allow": ["scenarios/web-console/**"],
  "acceptance_deny": ["scenarios/web-console/secrets/**"],
  "created": "ISO-8601",
  "updated": "ISO-8601"
}
```

`scope` is not a valid backlog field. Use `acceptance_allow` and `acceptance_deny` to describe execution boundaries, and use `initiative` to group related items.

#### `depends_on` (optional)

Array of `"kind/name"` strings referencing other backlog items this item depends on. Dependency validation rules:
- All referenced items must exist at creation time
- Circular dependencies are rejected (cycle detection via topological sort)
- Dependencies are enforced during batch-queue: items are queued in topological order so prerequisites run first

#### `acceptance_allow` (optional)

Array of glob patterns for file paths expected to be modified (e.g., `["scenarios/web-console/**", "packages/proto/**"]`). Defines the expected change boundaries — files matching these globs are expected to be modified during execution. Patterns starting with `scenarios/<name>/` also identify which scenarios are targeted for post-execution review.

#### `acceptance_deny` (optional)

Array of glob patterns for file paths that must NOT be modified (e.g., `["scenarios/web-console/secrets/**"]`). Used as sandbox acceptance denylist.

#### Acceptance Patterns: How They Work Together

- **`acceptance_allow`** defines the expected change boundaries — files matching these globs are expected to be modified during execution. Patterns starting with `scenarios/<name>/` also identify which scenarios are targeted for post-execution review.
- **`acceptance_deny`** defines forbidden change boundaries — files matching these globs must NOT be modified.
- **Post-execution review** uses these fields to validate agent work and identify target scenarios: modifications outside `acceptance_allow` are flagged as deviations, and modifications matching `acceptance_deny` are flagged as violations.

#### `initiative` (optional)

String label grouping this item with other items under a shared initiative. Used by the initiatives API (`/api/v1/initiatives`) to compute rollup status across member items.

### `plan.md`

Markdown implementation plan. This is the primary artifact that executing agents receive as context. The mandatory section structure, convergence patterns, quality gates, and guardrails are defined by the `implementation-plan-authoring` skill (`prompt-manager skill read implementation-plan-authoring`). Sections may be `<!-- TBD -->` until populated through workshop rounds.

### `handoff/` (idea only, generated during processing)

`handoff/` is not a workshop artifact. It is a derived execution package written by swarm-manager when an idea backlog item is processed. It exists to preserve the finalized backlog context when work is handed off to ecosystem-manager.

- `handoff/brief.md` — agent-facing execution brief; use as ecosystem-manager task notes
- `handoff/manifest.json` — machine-readable contract with provenance, boundaries, and resolved decisions
- `handoff/source-index.json` — pointers back to the source files used to derive the package

Do not manually maintain `handoff/` during workshop rounds. Update `plan.md` and workshop state; swarm-manager regenerates the handoff package from those authoritative sources when execution begins.

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
| `scope_defined` | No acceptance criteria set | Target area identified in description/plan but no acceptance_allow patterns | acceptance_allow defined, covers planned changes | Both acceptance_allow and acceptance_deny defined, plan changes align with globs |
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

For idea execution, `handoff/` is a derived transport artifact, not a competing planning authority. If `plan.md` changes, regenerate the handoff rather than editing the handoff by hand.

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

### List backlog items
```bash
swarm-manager backlog list [--kind <kind>] [--status <status>]
```

### Create a backlog item
```bash
swarm-manager backlog create --data '{"kind":"idea","name":"my-feature","title":"My Feature","description":"...","acceptance_allow":["scenarios/web-console/ui/**","scenarios/web-console/api/**"],"acceptance_deny":["scenarios/web-console/secrets/**"]}'
```

### Update a backlog item
```bash
swarm-manager backlog update --kind <kind> --name <name> --data '{"acceptance_allow":["scenarios/web-console/**"]}'
```

Notes:
- `backlog update` is a sparse patch. Omitted fields stay unchanged.
- Use empty strings to clear scalar fields like `description` or `initiative`.
- Use empty arrays to clear list fields like `tags`, `depends_on`, `acceptance_allow`, or `acceptance_deny`.

### Delete a backlog item
```bash
swarm-manager backlog delete --kind <kind> --name <name>
```

### Queue a single item
```bash
swarm-manager backlog queue --kind <kind> --name <name> [--execute] [--force] [--mode manual|scheduled|yolo]
```

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

### Research / Initialize / Workshop

Trigger agent-driven research, initialization, or workshop on a backlog item.

```bash
swarm-manager backlog research --kind <kind> --name <name> --data '{"mode":"<mode>"}'
```

**Modes:**

| Mode | Purpose |
|------|---------|
| `initialize` | Spawn initialization agent — creates first workshop round |
| `workshop` | Spawn workshop agent — advances to next workshop round |
| `research` / `explore` / `investigate` | Spawn research agent (aliases, all equivalent) |

**Optional `--data` JSON fields:**

| Field | Type | Purpose |
|-------|------|---------|
| `mode` | string | Research mode (see above) |
| `prompt` | string | Additional user context appended to the skill prompt |
| `project_root` | string | Override project root (default: ".") |
| `context_paths` | string[] | File paths to include as reference material |
| `context_target_ids` | string[] | Operational target IDs from archive |
| `context_requirement_ids` | string[] | Requirement IDs from archive |

### Batch create items
```bash
# Preview or create multiple items atomically (all-or-nothing). The request can
# assign each item to an initiative and can also create/update initiative metadata
# inline through the top-level `initiatives` array.
cat > /tmp/batch-items.json <<'EOF'
{
  "items": [
    {
      "kind": "fix",
      "name": "auth-bug",
      "title": "Fix auth bug",
      "description": "...",
      "initiative": "release-control",
      "acceptance_allow": ["scenarios/swarm-manager/api/**"]
    },
    {
      "kind": "idea",
      "name": "new-feature",
      "title": "New feature",
      "description": "...",
      "depends_on": ["fix/auth-bug"],
      "initiative": "release-control",
      "acceptance_allow": ["scenarios/web-console/**"]
    }
  ],
  "initiatives": [
    {
      "name": "release-control",
      "title": "Release Control",
      "description": "Shared release-governance improvements",
      "status": "active"
    }
  ]
}
EOF
swarm-manager backlog batch-create --file /tmp/batch-items.json --preview

# Create for real after previewing:
swarm-manager backlog batch-create --file /tmp/batch-items.json
```

Notes:
- `--preview` validates item payloads, dependency refs, and initiative actions without writing anything.
- Unknown fields are rejected. Do not send legacy `scope`.
- Initiative assignment is per item (`"initiative": "..."`), not a top-level CLI flag.

### Batch queue items
```bash
# Queue multiple items in dependency-safe topological order.
# Default is preview mode (dry run); add --execute to commit.
swarm-manager backlog batch-queue --items fix/auth-bug,idea/new-feature
swarm-manager backlog batch-queue --items fix/auth-bug,idea/new-feature --execute
swarm-manager backlog batch-queue --items fix/auth-bug,idea/new-feature --execute --force --mode yolo
```

### Captures commands
```bash
swarm-manager captures list                              # List all captures
swarm-manager captures create --text "Quick thought..."  # Create a capture from text
swarm-manager captures create --text "..." --file a.png  # Create with text and file attachment(s)
swarm-manager captures get --id <id>                     # Get a specific capture
swarm-manager captures delete --id <id>                  # Delete a capture
swarm-manager captures classify --id <id>                # AI-classify a capture into a backlog item
```

### Initiatives commands

Initiatives are stored as folders at `.vrooli/initiatives/{name}/` containing an `initiative.json` metadata file and any additional context files (strategy docs, decision logs, health reports, etc.).

```bash
swarm-manager initiatives list                                    # List all initiatives with rollup status
swarm-manager initiatives get --name <name>                       # Get initiative details and member items
swarm-manager initiatives create --data '{"name":"my-init","title":"My Initiative","description":"...","status":"active"}'
swarm-manager initiatives update --name <name> --data '{"title":"Updated Title"}' # Partial update
swarm-manager initiatives delete --name <name>                    # Delete an initiative
swarm-manager initiatives add-items --name <name> --items kind/name,kind/name   # Add items to initiative
swarm-manager initiatives remove-items --name <name> --items kind/name,kind/name # Remove items from initiative
```

### Initiative file commands

Initiatives support arbitrary context files alongside the `initiative.json` metadata. Use these commands to manage strategic context, decision logs, health reports, or any other files.

```bash
swarm-manager initiatives files --name <name>                              # List all files in an initiative
swarm-manager initiatives file-get --name <name> --path <path>             # Read a file
swarm-manager initiatives file-get --name <name> --path <path> --out local-file  # Download to local file
swarm-manager initiatives file-upload --name <name> --path <path> --stdin  # Upload from stdin (heredoc)
swarm-manager initiatives file-upload --name <name> --path <path> --file <local-file>  # Upload local file
swarm-manager initiatives file-upload --name <name> --path <path> --content "inline text"  # Upload inline
swarm-manager initiatives file-op --name <name> --op delete --source <path>  # Delete a file
swarm-manager initiatives file-op --name <name> --op rename --source <old> --dest <new>  # Rename
swarm-manager initiatives file-op --name <name> --op move --source <from> --dest <to>    # Move
swarm-manager initiatives file-op --name <name> --op copy --source <from> --dest <to>    # Copy
```

Notes:
- `initiative.json` is protected and cannot be modified through file operations (use `initiatives update` instead).
- The `--stdin` flag is preferred for content with special characters (avoids shell quoting issues).
- File paths are relative to the initiative folder (e.g., `decisions/d1.md`, `strategy.md`).

### Overview command
```bash
swarm-manager overview                  # Aggregated view (backlog counts, initiatives, dep graph, stats)
swarm-manager overview --format json    # JSON output
swarm-manager overview --format markdown # Markdown output (default)
```

### Agent-manager run commands
```bash
swarm-manager agent-manager run-get --id <run-id>   # Get execution run status
swarm-manager agent-manager run-stop --id <run-id>  # Stop a running execution
```

### Stats commands
```bash
swarm-manager stats summary                # Full stats dashboard (throughput, timing, blocking, initiatives, agent efficiency)
swarm-manager stats throughput             # Throughput metrics (completed/created counts, net delta)
swarm-manager stats blocking              # Blocking analysis (blocked ratio, top reasons, avg block hours)
swarm-manager stats initiatives           # Initiative health (per-initiative completed/total/blocked/scope_creep)
swarm-manager stats agent                 # Agent efficiency (success/failure rates, execution time, workshop rounds)
```

The `summary` subcommand returns all stat categories in one call. Use the focused subcommands when you only need a specific slice.

### Convert item kind
```bash
swarm-manager backlog convert --kind <kind> --name <name> --target-kind <new-kind>
```

### Export / Import backlog
```bash
swarm-manager backlog export [--kind <kind>] [--name <name>]
swarm-manager backlog import --file <file>
```

### Prompt trace
```bash
swarm-manager backlog prompt-trace --kind <kind> --name <name>
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
