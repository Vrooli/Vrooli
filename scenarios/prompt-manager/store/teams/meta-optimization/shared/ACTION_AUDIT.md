# Action Audit - Rolling

Action health and adoption snapshot. Maintained by `skill-optimizer`.

## Baseline - 2026-08-09

| Action | Status | Validation | Discoverability | Disposition | Notes |
|--------|--------|------------|-----------------|-------------|-------|
| action:scenario.status.show | active | valid; dry-run passes with `--scenario=prompt-manager` | discoverability verified by `prompt-manager action list`; live registry entry | adopted | Active read-only seed Action; owner `project:vrooli`. |
| action:team.swarm.work.list | missing | 404 on show/validate/run | absent from `prompt-manager action list` | stale-register | Previously recorded seed is not in the live Action registry; do not count it as active or propose consumers until reintroduced and revalidated. |
| agent-system.framework-health | active | valid; dry-run passes (`prompt-manager graph audit`) | exact match found by `prompt-manager discover` | existing-action-reference | Read-only framework-health sensor Action owned by `scenario:prompt-manager`; API-read permission; runnable. Validation has an owner-governance warning because the owning scenario lacks declared `cli/manifest.json` governance. Proposed use: replace manual framework-health collection references in `agent-system-audit`; no new Action needed. |
| skill-validation-contract-audit | no exact Action | blocked | discovery returned skill guidance, not an Action | capability-work-item | No governed one-command Action currently owns skill command/reference validation; backlog `fix-skill-validation-current-cli-contracts` records the required owner work. |
| search-hub query | no exact Action | blocked | `prompt-manager discover "search-hub query" --type action` returned unrelated Actions only | capability-work-item | Querying remains a scenario CLI/program surface, not a registered Action. Existing backlog `search-hub-interactive-readiness` is the owner-routed substrate work; do not add a wrapper without a stable single-command contract and baseline. |
| documentation-health audit | no exact Action | blocked | `prompt-manager discover "audit documentation health and traceability" --type all` found no docs-health Action; only `agent-system.framework-health`, `bas.audit`, and unrelated skills | cli-backlog | The workflow spans `knowledge-observatory docs audit/health` and `plan-manager` authoring context. Existing owner item `documentation-health-command-drift` repairs the stale syntax and evaluates stable promotion; no duplicate Action proposal. |

## Measurement Signals

- Action count by status: 1 active, 1 stale register entry.
- Active runnable Action count: 1.
- Validation failures: 0 in targeted seed validation.
- Graph inbound warnings: 0 for the one registered seed; the missing `team.swarm.work.list` entry has no live contract to validate.
- Run history signals: no post-adoption usage baseline yet.
- Skill prose collapsed to Action references: 0.
- Skill prose collapsed to Action references: 1 proposed (`agent-system-audit` → `agent-system.framework-health`); acceptance pending owner work item.
- Repeated manual operation count from run-introspector: not yet measured.
- 2026-08-29 discovery for visited-tracker coverage found no exact Action; retain as skill-improvement work, not an Action candidate.

## Revisit Queue

1. After four meta-optimization heartbeats, compare Action discoveries/runs against repeated manual operations.
2. Continue adding seed Actions only when one stable Vrooli-controlled CLI command owns the operation.

## 2026-09-08 Rotation Addendum

| Operation | Status | Validation | Discoverability | Disposition | Notes |
|-----------|--------|------------|-----------------|-------------|-------|
| audit-scope session constraints | no exact Action | not applicable | Discovery returned no exact Action; closest `agent-system.framework-health` owns framework-health collection, not session scoping | no-action | The skill is a compact judgment/permission contract with no deterministic operation, stable CLI owner, or input/output contract. Do not propose an Action until a governed audit-scope operation exists. |

## 2026-09-09 Rotation Addendum

| Operation | Status | Validation | Discoverability | Disposition | Notes |
|-----------|--------|------------|-----------------|-------------|-------|
| observability / telemetry wiring guidance | no exact Action | blocked | `prompt-manager discover "observability telemetry wiring instrumentation" --type all` returned `signal-and-feedback-surface-design` and readiness skills, but no Action | capability-work-item | Existing `signal-and-feedback-surface-design` is judgment-only and has no stable single-command owner, input/output contract, or permission surface for an Action. Do not invent a wrapper. Guide G32 is tracked by backlog `chore/add-observability-telemetry-guide-capability`; reassess for an Action only after an owning CLI/program exists. |

## 2026-09-10 Rotation Addendum

| Operation | Status | Validation | Discoverability | Disposition | Notes |
|-----------|--------|------------|-----------------|-------------|-------|
| canonical in-place file edit outside a sandbox (Act A25) | no exact Action | blocked | `prompt-manager discover "edit a file in place outside a sandbox" --type all` found only `sandbox` and unrelated commit/provenance skills; no executable Action | capability-work-item | Coverage `act/A25` is missing with no runtime owner/provider. Existing `workspace-sandbox` exposes create/diff/promote, not canonical edits. Filed `fix/capability-act-a25-canonical-file-edit-owner-20260910`; runEligible=false until an owner, contract, permissions, and refusal tests exist. No Action candidate proposed. |

## 2026-09-11 Rotation Addendum

| Operation | Status | Validation | Discoverability | Disposition | Notes |
|-----------|--------|------------|-----------------|-------------|-------|
| storage architecture validation / isolation proof | programmatic CLI, no exact Action | CLI help available; `storage-manager validate scenario prompt-manager --json` executed and returned structured findings; no Action contract | `prompt-manager discover "validate scenario storage architecture and isolation with storage-manager" --type all` returned skills only; `action show/validate storage-manager.storage.validate` returned not found | graduate-reference | `storage-manager` is the validated programmatic home for `storage-steer` detection, so no duplicate Action candidate is proposed. The CLI contract is scenario-name input with structured analyzer findings/output; permissions and Action receipt semantics are not registered. Revisit if repeated manual invocation justifies a governed Action. |
