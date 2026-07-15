## Swarm Manager Recommendations

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — canonical reference for folder structure, artifact schemas, and interaction patterns.

Use `swarm-manager` as the single control plane for scenario-change intent.
This skill is mandatory for teams that produce recommendations/findings for execution.

---

### 2. Team-to-Backlog Contract

- Scenario Feature Team -> `idea` or `execute`
- Scenario QA Team -> `fix` or `execute`

Do not directly edit target-scenario code as part of this workflow.

---

### 3. Required Fields

Every item must include these fields in `spec.json`, `notes.md`, or attached evidence files:

- `targetScenario`
- `problemOrOpportunity`
- `proposedAction`
- `evidence`
- `riskLevel`
- `executionModeHint`
- `createdByTeam`
- `sourceRunId`

Recommended `riskLevel` values: `low`, `medium`, `high`, `critical`.
Recommended `executionModeHint` values: `manual`, `scheduled`, `yolo`.

#### Description Quality Standard

The `description` field in `spec.json` is what agents see first. Write it for a reader with **zero context** about what just happened. Never echo a raw score or tool output as the description.

A good description includes:
1. **What** is wrong — specific dimension, category, and affected files
2. **Where** — top 3-5 file paths with the worst violations
3. **Example** — one concrete violation (e.g., "function X has cyclomatic complexity 47")
4. **Target** — measurable success criteria (e.g., "bring score from 0 to ≥70")
5. **Reproduce** — the command to see the issue firsthand

Bad: `"GCT code quality score 0 with 151 violations (complex_functions, long_files, lint_issues)."`

Good: `"deployment-manager code quality score 0/100 with 151 violations. Worst offenders: api/orchestrator.go (complexity 47 in deployRelease, 1200 lines), api/handler.go (890 lines, 12 lint issues), ui/src/components/DeployFlow.tsx (complexity 22). Target: score ≥70. Reproduce: git-control-tower review-run deployment-manager --json"`

---

### 4. Authoring Pattern

1. Build a stable backlog name (kebab-case, deterministic).
2. Create backlog item.
3. Attach deep evidence in files.  
That's it! Don't queue any tasks - all backlog execution is controlled by swarm-manager settings you're not allowed to modify.

Example (debug team -> `fix`):

```bash
swarm-manager backlog create --data '{
  "kind":"fix",
  "name":"scenario-x-auth-timeout-regression",
  "title":"Fix auth timeout regression",
  "description":"Intermittent timeout during token refresh blocks login.",
  "priority":2,
  "tags":["scenario-x","auth","regression","debug"],
  "status":"ready"
}'
```

---

### 5. Evidence Packaging

For each backlog item folder, include:

- `notes.md`: **self-contained briefing document** (see requirements below)
- `evidence/` files: logs, stack traces, links, metrics snapshots
- `tests.md` (optional): recommended validation and acceptance checks

Keep evidence factual and reproducible.

#### notes.md Must Stand Alone

`notes.md` is the primary briefing for the execution agent. It must be understandable **without opening any JSON or evidence files**. Extract the key data from evidence files into notes.md directly.

Required sections:
- **Problem**: What is wrong, with specific file paths and violation details
- **Top Violations**: The 5-10 worst offenders listed by file path, category, and count — extracted from evidence, not just a reference to a JSON file
- **Impact**: What downstream work or quality is affected
- **Reproduction**: Exact command(s) to observe the issue
- **Success Criteria**: Concrete, measurable definition of done (target scores, violation counts, passing tests)
- **Proposed Action**: Ordered steps prioritized by impact, referencing specific files and functions

---

### 7. Verification Commands

```bash
swarm-manager backlog get --kind "<kind>" --name "<name>"
swarm-manager execution list --backlog-kind "<kind>" --backlog-name "<name>"
swarm-manager execution get --id "<execution-id>"
```
