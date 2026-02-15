## Swarm Manager Recommendations

Use `swarm-manager` as the single control plane for scenario-change intent.
This skill is mandatory for teams that produce recommendations/findings for execution.

---

### 2. Team-to-Backlog Contract

- Scenario Debug Team -> `fix`
- Scenario Feature Team -> `idea` or `execute`
- Scenario QA Team -> `fix` or `execute`
- Scenario Refactor Team -> `execute` or `fix`

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

---

### 4. Authoring Pattern

1. Build a stable backlog name (kebab-case, deterministic).
2. Create backlog item.
3. Attach deep evidence in files.  
That's it! Don't queue any tasks - all backlog execution is controlled by swarm-manager settings you're not allowed to modify.

Example (debug team -> `fix`):

```bash
swarm-manager backlog create '{
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

- `notes.md`: concise executive summary + proposed action
- `evidence/` files: logs, stack traces, links, metrics snapshots
- `tests.md` (optional): recommended validation and acceptance checks

Keep evidence factual and reproducible.

---

### 7. Verification Commands

```bash
swarm-manager backlog get <kind> <name>
swarm-manager execution list --backlog-kind <kind> --backlog-name <name>
swarm-manager execution get <execution-id>
```
