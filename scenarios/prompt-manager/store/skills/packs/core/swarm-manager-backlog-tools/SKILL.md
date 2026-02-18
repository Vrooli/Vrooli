# Backlog Tools

## Purpose

Canonical reference for the swarm-manager backlog item data model and interaction patterns. All skills that read or write backlog item artifacts should defer to this document for folder structure, artifact schemas, reading order, CLI commands, and mutation rules.

**Scope boundaries:**
- **In scope**: folder structure, artifact schemas, reading order, CLI commands, mutation rules, troubleshooting
- **Out of scope**: processing behavior, completion summaries, verification checklists (see `swarm-manager-processing-guidance`)

## Folder Structure

Every backlog item lives at `{{ITEM_FOLDER}}` and follows this layout.

> **Filesystem path:** Backlog items are plain directories stored at `scenarios/swarm-manager/{ideas|research|fix|execute}/{item-name}/`. The `{{ITEM_FOLDER}}` variable resolves to this absolute path at runtime when invoked through the swarm-manager workflow. Agents that know the item kind and name can read/write files directly on the filesystem as an alternative to the CLI commands below.

```
item-folder/
├── spec.json              # Item metadata (kind, title, description, status)
├── clarify/
│   └── questions.json     # Clarifying Q&A (if clarify workflow ran)
├── suggest/
│   └── suggestions.json   # Suggestions with accept/reject (if suggest ran)
├── enhance/
│   ├── summary.md             # Refined plan (if enhance ran)
│   ├── prd-context.md         # PRD context brief for prd-control-tower (if enhance ran)
│   ├── requirements-context.md # Requirements context (if relevant source material exists)
│   └── doc-outlines.md        # Documentation outlines (if relevant source material exists)
├── research/
│   └── summary.md         # Research findings (if deep research ran)
├── archive/
│   └── ...                # Superseded artifacts from previous runs
└── [user files]           # Any additional context added by user
```

### Subfolder Reference

| Subfolder | Creator | Purpose |
|-----------|---------|---------|
| `clarify/` | clarify agent | Stores clarifying questions and user answers |
| `suggest/` | suggest agent | Stores improvement suggestions and user decisions |
| `enhance/` | enhance agent | Stores the refined plan (`summary.md`) and staging artifacts for the process step (`prd-context.md`, `requirements-context.md`, `doc-outlines.md`) |
| `research/` | research agent | Stores feasibility research and findings |
| `archive/` | user / system | User-provided materials (prior scenario artifacts, requirements, designs) and superseded artifacts from previous workflow runs. Agents should read but not modify. |
| root | user / system | `spec.json` metadata, user-uploaded context files |

## Artifact Schemas

### `spec.json`

```json
{
  "kind": "idea | fix | execute",
  "name": "kebab-case-name",
  "title": "Human-readable title",
  "description": "Full description of the item",
  "priority": 1,
  "status": "draft | ready | in-progress | done | blocked",
  "tags": ["tag1", "tag2"],
  "created_at": "ISO-8601",
  "updated_at": "ISO-8601"
}
```

### `clarify/questions.json`

```json
{
  "questions": [
    {
      "id": "q1",
      "question": "What authentication method should be used?",
      "category": "technical",
      "importance": "critical",
      "options": ["OAuth 2.0", "JWT tokens", "Session-based"],
      "answer": ""
    }
  ],
  "generated_at": "ISO-8601",
  "max_questions": 10
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier (q1, q2, ...) |
| `question` | Yes | Clear, specific question ending with ? |
| `category` | Yes | One of: users, technical, scope, constraints, integration |
| `importance` | Yes | One of: critical, important, nice-to-have |
| `options` | No | Suggested answers (2-4 options, include "Other" if open-ended) |
| `answer` | Yes | Empty string (filled by user later) |

### `suggest/suggestions.json`

```json
{
  "suggestions": [
    {
      "id": "s1",
      "suggestion": "Use WebSocket instead of polling for real-time updates",
      "details": "WebSocket would reduce latency from seconds to milliseconds...",
      "category": "architecture",
      "impact": "high",
      "status": "pending",
      "rejection_reason": ""
    }
  ],
  "generated_at": "ISO-8601",
  "max_suggestions": 7
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier (s1, s2, ...) |
| `suggestion` | Yes | One-line summary of the improvement |
| `details` | Yes | Full explanation with rationale (2-4 sentences) |
| `category` | Yes | One of: architecture, ux, scope, risk, opportunity |
| `impact` | Yes | One of: high, medium, low |
| `status` | Yes | "pending" for new; "accepted" or "rejected" after user review |
| `rejection_reason` | Yes | Empty string (filled by user if rejected) |

### `enhance/` folder

The `enhance/` folder serves dual purposes: the refined plan and staging artifacts for the process step.

| File | Purpose |
|------|---------|
| `summary.md` | Refined plan — the source of truth for what to implement. Sections: Overview, Clarifications Applied, Suggestions Integrated, Refined Scope, Implementation Notes, Success Criteria, Readiness Gate, Staging Artifacts Produced. |
| `prd-context.md` | Synthesized PRD context brief ready for `prd-control-tower prd generate` consumption. Combines all backlog sources into the free-form brief format. |
| `requirements-context.md` | Requirements context ready for `prd-control-tower requirements generate`. Present only when requirements-related source material exists. |
| `doc-outlines.md` | Documentation outlines (README sections, RESEARCH findings, PROBLEMS entries). Present only when documentation-related source material exists. |

See `swarm-manager-enhance-idea` for the full template and staging artifact guidelines.

### `research/summary.md`

Markdown document with sections: Executive Summary, Feasibility Assessment, Dependency Analysis, Implementation Approaches, Risks & Mitigations, Next Steps. See the relevant research skill for the full template.

## Source Authority Hierarchy

The backlog folder represents a refinement pipeline. Each stage builds on the previous, producing progressively more refined and authoritative output. Understanding this hierarchy is essential for every agent that reads or writes backlog artifacts.

### Refinement Levels

```
Most refined / highest authority
  ┌─────────────────────────────────────────────────────┐
  │  enhance/          Synthesized plan + staging        │
  │                    artifacts. The "compiled" output   │
  │                    of everything below it.            │
  ├─────────────────────────────────────────────────────┤
  │  clarify/ + suggest/   User decisions — answered     │
  │                    questions and accepted/rejected    │
  │                    suggestions. Direct user intent.   │
  ├─────────────────────────────────────────────────────┤
  │  research/         Advisory findings — informs but   │
  │                    does not override user decisions.  │
  ├─────────────────────────────────────────────────────┤
  │  spec.json         Original description and metadata.│
  │                    Superseded by enhance/ when it     │
  │                    exists.                            │
  ├─────────────────────────────────────────────────────┤
  │  archive/          Raw materials from a prior or     │
  │                    existing scenario. Least refined   │
  │                    — used as source material, not     │
  │                    as authoritative specification.    │
  └─────────────────────────────────────────────────────┘
Least refined / lowest authority
```

**Key principle:** `enhance/` is the most up-to-date and authoritative source. It represents the fully synthesized output of all refinement stages. When `enhance/` exists, treat it as the primary source of truth. When it doesn't, reconstruct the specification from the lower-authority sources.

### How Refinement Accumulates

The pipeline is iterative. Users may run clarify → suggest → enhance, then go back, add more questions or suggestions, and enhance again. Each pass builds on the previous:

- **First enhance run**: Reads spec, clarify, suggest, research, archive → produces `enhance/summary.md` + staging artifacts
- **Subsequent clarify/suggest runs**: Read the existing `enhance/` output as context, generating questions/suggestions that refine what's already been synthesized
- **Subsequent enhance runs**: Read the prior `enhance/` output as a **foundation**, then layer on new clarify answers, new accepted suggestions, and any new research — producing an updated synthesis

This means `enhance/` is never stale in the way `archive/` can be. Each enhance run incorporates everything that came before it. Agents re-running any pipeline step should always read `enhance/` (if it exists) as their starting context.

### Reading Order

When processing a backlog item, read artifacts in this order:

1. `enhance/summary.md` — if it exists, start here (it's the most refined source of truth)
   - Also check for staging artifacts: `enhance/prd-context.md`, `enhance/requirements-context.md`, `enhance/doc-outlines.md`
2. `spec.json` — item metadata; description is superseded by enhance/ when it exists
3. `clarify/questions.json` — review answered questions (may contain new answers since last enhance run)
4. `suggest/suggestions.json` — review accepted/rejected suggestions (may contain new decisions since last enhance run)
5. `research/summary.md` — advisory feasibility findings
6. `archive/` — raw materials; only use for content not already captured in enhance/
7. User-uploaded files — additional context

### Decision Authority Rules

When sources conflict, apply this precedence (highest to lowest):

1. **Answered questions** in `clarify/questions.json` are **definitive** — always implement as answered, even if they contradict a prior enhance run
2. **Accepted suggestions** in `suggest/suggestions.json` **must** be incorporated
3. **Rejected suggestions** must **NOT** be implemented
4. **`enhance/`** supersedes `spec.json` and `archive/` — it was synthesized later with more context
5. **Research findings** are advisory — they inform but do not override user decisions
6. **`archive/`** is raw source material — use it only when `enhance/` doesn't cover the same content

> **Why answered questions outrank enhance/:** A user may answer new questions *after* a prior enhance run. Those new answers represent the latest user intent and must take precedence, even if the current enhance/summary.md doesn't yet reflect them. The next enhance run will incorporate them.

## CLI Commands

### Read item metadata
```bash
swarm-manager backlog get <kind> <name>
```

### List files in item folder
```bash
swarm-manager backlog files <kind> <name>
```

### Read a specific file
```bash
swarm-manager backlog file-get <kind> <name> <relative-path>
# Example: swarm-manager backlog file-get idea my-feature clarify/questions.json
```

### Upload a file
```bash
swarm-manager backlog file-upload <kind> <name> <relative-path> <content>
# Example: swarm-manager backlog file-upload idea my-feature clarify/questions.json '{"questions":[]}'
```

## Mutation Rules

| Artifact | Who may write | Who may read |
|----------|--------------|--------------|
| `spec.json` | system, user, enhance agent | all agents |
| `clarify/questions.json` | clarify agent (questions), user (answers) | all agents |
| `suggest/suggestions.json` | suggest agent (suggestions), user (decisions) | all agents |
| `enhance/*` | enhance agent | all agents |
| `research/summary.md` | research agent | all agents |
| `archive/*` | user, system (when archiving scenario artifacts) | all agents (read-only) |
| user files (root) | user | all agents |

## Troubleshooting & Edge Cases

### 404 on `file-get`
The artifact hasn't been created yet. This is normal — not every workflow step runs. Check which workflows have completed by listing files with `backlog files`.

### Empty `archive/` folder
No previous runs have been superseded. The archive folder is created on demand when an agent re-runs a workflow step.

### Unanswered questions in `clarify/questions.json`
Questions with an empty `answer` field have not been answered by the user. Do not assume answers — treat unanswered questions as open unknowns and flag them if they are critical.

### Missing `enhance/summary.md`
The enhance workflow hasn't run. Fall back to `spec.json` description plus any answered questions, accepted suggestions, and raw `archive/` materials as the working specification. See the source authority hierarchy for full fallback rules.

### Conflicting information
Apply the source authority hierarchy: answered questions > accepted suggestions > enhance/ > spec.json > archive/. See the "Decision Authority Rules" section for details and the rationale for why new answered questions outrank a prior enhance run.
