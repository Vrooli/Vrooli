# Scenario QA — Plan of Record

This folder is the **plan-of-record** for Vrooli's scenario-quality discipline: structural quality audits, programmatic readiness reviews, root-cause bug investigation, and contrarian challenge of QA outcomes. It's maintained by the `scenario-qa` team and consumed by its members every heartbeat.

The team's live operating rules are at [`scenarios/prompt-manager/store/teams/scenario-qa/shared/TEAM.md`](../../scenarios/prompt-manager/store/teams/scenario-qa/shared/TEAM.md). This folder is the strategic-canon side; that file is the runtime contract.

See [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../agent-system/TEAM_DOCS_PATTERNS.md) for the pattern definition.

## Start here for agents

Use this README first, then choose the file or sub-hub that matches the work:

| Question | Start with |
|---|---|
| How do I report a bug I just observed? | [`BUG_REPORT_TAXONOMY.md`](BUG_REPORT_TAXONOMY.md) — invoke `prompt-manager skill read report-bug` |
| Which investigation method applies to this bug? | [`investigation-techniques/README.md`](investigation-techniques/README.md) |
| Which audit lens applies to this scenario? | [`audit-techniques/README.md`](audit-techniques/README.md) |
| Which programmatic readiness check applies? | [`readiness-checks/README.md`](readiness-checks/README.md) |
| What is the team's mission and member roster? | this README §"Team shape" + `shared/TEAM.md` |
| How does a bug enter scenario-qa? | this README §"Cross-team flow" |

## Team shape

scenario-qa runs four members:

| Member | Role | Primary signal |
|---|---|---|
| `programmatic-qa-runner` | GCT-driven readiness reviews; creates fix/chore/execute backlog items | `qa-run/<scenario-id>` knowledge entries |
| `quality-auditor` | Judgment-based structural audits using a rotation of seven audit lenses | `topic[example]:quality-audit/<scenario-id>/<skill-id>` knowledge entries |
| `bug-investigator` | Drains `bug-inbox/*` (universal-source), applies investigation techniques, writes audit log | `bug-investigation-report/<slug>` knowledge entries |
| `qa-contrarian` | Challenges QA outcomes (audits, investigations, readiness backlog) — surfaces gaps in reasoning | `challenge-report/*` knowledge entries |

Decision contexts owned by the team:

| Context | Owner | Purpose |
|---|---|---|
| `preemptive-qa-backlog` | programmatic-qa-runner | GCT-driven QA findings → Swarm Manager fix/chore/execute backlog items |
| `quality-audit-backlog` | quality-auditor | Judgment-based structural audit findings → Swarm Manager execute backlog items |
| `bug-resolution-proposal` | bug-investigator | Cross-cutting fixes that require operator approval (e.g., rename a CLI verb because three bugs trace to its ambiguous name) |

## Folder map

| File / folder | Purpose |
|---|---|
| [`BUG_REPORT_TAXONOMY.md`](BUG_REPORT_TAXONOMY.md) | Human-readable view of `bug-report-taxonomy.json`. Signal types, schemas, action-selection rules, evidence rules. |
| [`bug-report-taxonomy.json`](bug-report-taxonomy.json) | Machine-readable taxonomy sidecar (loaded by the heartbeat builder; cited by `bug-investigator/topics.json`). |
| [`investigation-techniques/`](investigation-techniques/) | Strategic canon for techniques the bug-investigator applies. One paired doc + skill per technique (mirrors `path:docs/marketing/post-techniques/`). |
| [`audit-techniques/`](audit-techniques/) | Strategic canon for `quality-auditor`'s seven audit lenses. Same paired doc + skill discipline. |
| [`readiness-checks/`](readiness-checks/) | Strategic canon for `programmatic-qa-runner`'s individual readiness dimensions. Stub — populated as GCT dimensions stabilize. |

## Cross-team flow

scenario-qa is the bug-triage hub of the agent system. The `bug-inbox/*` topic prefix is **universal-source**: any team's members may write to it, by invoking the `report-bug` skill. The `bug-investigator` drains the inbox; classification is deterministic-prefix (signal type embedded in the topic), so no separate classifier skill is needed — investigation includes classification as its first step.

**Sister flow.** The agent system has a second universal observation flow at `topic:friction-inbox/*`, drained by `friction-curator`, fed by the `report-friction` skill — for system-level capture-leak (tooling gaps, run-execution friction, storage-map confusion, recurring workarounds). Use `report-bug` for broken code or scenario behavior; use `report-friction` for things that worked but were harder than they should have been. See [`docs/meta-optimization/FRICTION_REPORT_TAXONOMY.md`](../meta-optimization/FRICTION_REPORT_TAXONOMY.md).

```
any-team/* ─[report-bug skill]──▶ scenario-qa/bug-inbox/<signal-type>/<slug>
                                          │
                                          ▼
                              scenario-qa/bug-investigator
                                          │
                ┌─────────────────────────┼─────────────────────────┐
                ▼                         ▼                         ▼
       bug-investigation-report/<slug>  swarm-manager/backlog    decision: bug-resolution-proposal
       (audit log; append-only)  (fix/chore items)        (operator review for cross-cutting fixes)
```

The `qa-contrarian` reads peer-team decisions and member outputs (including bug-investigation entries and audit findings) and writes `challenge-report/*` plus `challenge-resolution-record/*` entries — challenge to QA outcomes is a first-class output, not a side-channel comment. The lifecycle follows `docs/agent-system/CONTRARIAN_REVIEW.md`: reports are append-only evidence, resolution records carry latest state, and quiet reviews write no challenge report.

## Editing rules

- **Agents never write to these files directly.** All edits come through operator-approved decisions.
- **Edit context:** `bug-resolution-proposal` covers `BUG_REPORT_TAXONOMY.md` updates and bug-report-taxonomy schema changes; `meta-self-improvement` (on `meta-optimization`) covers new technique entries in `investigation-techniques/` and `audit-techniques/`; per-rotation skill updates go through the standard skill-edit decision flow.
- **Operator executes edits** on decision acceptance. Commit messages cite the decision id.
- **Drafts are not canon.** Synthesis-in-flux content lives elsewhere (working notebooks or `path:docs/agent-system/drafts/`); files in this folder are stable PoR.

## Doc + paired skill discipline

All three technique registries (`investigation-techniques/`, `audit-techniques/`, `readiness-checks/`) follow the same mandatory rule from [`docs/marketing/post-types/README.md`](../marketing/post-types/README.md):

> Every entry ships as `doc + paired skill`. This is a hard rule, not a recommendation. Neither half is optional, and neither half replaces the other. The doc holds *reasoning*; the skill holds *procedure*. A doc with no skill is a stale shrine. A skill with no doc is brittle.

Enforced by the canon coherence test at `scenarios/prompt-manager/test/agent_system_canon_test.sh`.

## Cross-references

- `docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md` — the 9-layer model the audit skill uses to evaluate scenario-qa members. The `skillless canon` smell motivated the audit-techniques registry's seven paired PoR docs.
- `docs/agent-system/INTAKE_PIPELINE.md` — the inbox-router-drain pattern used by the bug-investigator.
- `docs/agent-system/TOPICS.md` — registry of every active topic prefix; scenario-qa entries live there.
- `docs/agent-system/TOPICS_SCHEMA.md` — schema reference for `topics.json`; documents the `source_team: "*"` (universal-source) semantics that bug-inbox uses.
- `path:docs/marketing/post-techniques/README.md` — the gold-standard reference this folder's three registries replicate.
- `path:scenarios/swarm-manager/` — downstream consumer of the team's backlog items.

## Future PoR work

Flagged here so future operator-curated decisions can promote them when the substrate calls for it:

- **Quality principles.** A `PRINCIPLES.md` codifying the team's quality philosophy (behavior-oriented evidence, root-cause-over-symptom, contrarian-by-default). Workshop-pending; a few iterations of contrarian challenge-notes will surface the right principles.
- **Scenario-classification heuristics.** A `SCENARIO_CLASSES.md` declaring how scenarios are bucketed for QA priority (revenue-critical, capability-uplift, internal-tool, archived). Drives queue policy for both readiness reviews and quality audits.
- **`topic[future]:qa-inbox/*` and `topic[future]:audit-inbox/*` operator-fed inboxes.** Today there is no producer for these prefixes; adding them would create `orphan_input`. If `vision-walk-prep` or another future producer adds them as output, the `programmatic-qa-runner` and `quality-auditor` members gain an intake to drain.
- **Full readiness-checks registry.** Stub README only today; entries graduate once GCT readiness dimensions stabilize.
- **Future investigation techniques.** `scientific-debugging` is the only registered technique at landing time. Candidates surfaced by the bug-investigator's audit log: bisect-debugging, minimal-reproduction, differential-trace, comparative-environments, 5-whys, fishbone analysis. Each enters via `meta-self-improvement` decision.
- **Future audit techniques.** Beyond the seven existing skills: performance-audit, security-audit, deprecation-audit, accessibility-audit, observability-audit. Same graduation flow.
