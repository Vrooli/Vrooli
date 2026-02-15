# Backlog Tools

## Purpose

Canonical reference for the swarm-manager backlog item data model and interaction patterns. All skills that read or write backlog item artifacts should defer to this document for folder structure, artifact schemas, reading order, CLI commands, and mutation rules.

**Scope boundaries:**
- **In scope**: folder structure, artifact schemas, reading order, CLI commands, mutation rules, troubleshooting
- **Out of scope**: processing behavior, completion summaries, verification checklists (see `swarm-manager-processing-guidance`)

## Folder Structure

Every backlog item lives at `{{ITEM_FOLDER}}` and follows this layout:

```
item-folder/
├── spec.json              # Item metadata (kind, title, description, status)
├── clarify/
│   └── questions.json     # Clarifying Q&A (if clarify workflow ran)
├── suggest/
│   └── suggestions.json   # Suggestions with accept/reject (if suggest ran)
├── enhance/
│   └── summary.md         # Refined plan (if enhance ran)
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
| `enhance/` | enhance agent | Stores the refined, synthesized plan |
| `research/` | research agent | Stores feasibility research and findings |
| `archive/` | any agent | Superseded artifacts from previous workflow runs |
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

### `enhance/summary.md`

Markdown document with sections: Overview, Clarifications Applied, Suggestions Integrated, Refined Scope, Implementation Notes, Success Criteria. See `swarm-manager-enhance-idea` for the full template.

### `research/summary.md`

Markdown document with sections: Executive Summary, Feasibility Assessment, Dependency Analysis, Implementation Approaches, Risks & Mitigations, Next Steps. See the relevant research skill for the full template.

## Reading Order & Decision Hierarchy

When processing a backlog item, read artifacts in this order:

1. `spec.json` — understand the item's kind, title, description, and status
2. `clarify/questions.json` — review all answered questions
3. `suggest/suggestions.json` — review accepted/rejected suggestions
4. `enhance/summary.md` — read the refined plan (supersedes raw description)
5. `research/summary.md` — review any feasibility findings
6. User-uploaded files — additional context

### Decision Authority Rules

- **Answered questions** in `clarify/questions.json` are **definitive** — implement as answered
- **Accepted suggestions** in `suggest/suggestions.json` **must** be incorporated
- **Rejected suggestions** must **NOT** be implemented
- **Enhanced description** in `enhance/summary.md` **supersedes** the original description in `spec.json`
- **Research findings** are advisory — they inform but do not override user decisions

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
| `enhance/summary.md` | enhance agent | all agents |
| `research/summary.md` | research agent | all agents |
| `archive/*` | any agent (when superseding artifacts) | all agents |
| user files (root) | user | all agents |

## Troubleshooting & Edge Cases

### 404 on `file-get`
The artifact hasn't been created yet. This is normal — not every workflow step runs. Check which workflows have completed by listing files with `backlog files`.

### Empty `archive/` folder
No previous runs have been superseded. The archive folder is created on demand when an agent re-runs a workflow step.

### Unanswered questions in `clarify/questions.json`
Questions with an empty `answer` field have not been answered by the user. Do not assume answers — treat unanswered questions as open unknowns and flag them if they are critical.

### Missing `enhance/summary.md`
The enhance workflow hasn't run. Fall back to `spec.json` description plus any answered questions and accepted suggestions as the working specification.

### Conflicting information
If `enhance/summary.md` and `spec.json` disagree, `enhance/summary.md` wins (it was synthesized later with more context). If user answers contradict suggestions, user answers win.
