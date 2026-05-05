# Scenario QA Team

## Mission
Ensure scenario quality through structural quality audits, programmatic readiness reviews, root-cause bug investigation, and contrarian challenge of QA outcomes — producing evidence-rich backlog items, audit logs, investigation logs, and challenge notes that downstream agents act on.

## Scope
Owns scenario-quality findings (programmatic + structural), bug-triage for the entire agent system, and contrarian challenge of every scenario-qa output. Drains the universal-source `bug-inbox/*` topic that any team's members may write via the `report-bug` skill.

Does not directly edit target scenarios. Does not own monetization, marketing, infrastructure, or meta-layer optimization.

## Plan of Record
Strategic canon lives at [`docs/scenario-qa/README.md`](../../../../../../docs/scenario-qa/README.md), with three paired-doc-and-skill registries:
- [`path:docs/scenario-qa/investigation-techniques/`](../../../../../../docs/scenario-qa/investigation-techniques/README.md) — bug-investigator's methods (1 entry: `scientific-debugging`).
- [`path:docs/scenario-qa/audit-techniques/`](../../../../../../docs/scenario-qa/audit-techniques/README.md) — quality-auditor's seven lenses.
- [`path:docs/scenario-qa/readiness-checks/`](../../../../../../docs/scenario-qa/readiness-checks/README.md) — programmatic-qa-runner's individual checks (stub; populated as GCT dimensions stabilize).

Bug taxonomy: [`docs/scenario-qa/BUG_REPORT_TAXONOMY.md`](../../../../../../docs/scenario-qa/BUG_REPORT_TAXONOMY.md) (paired with `bug-report-taxonomy.json`).

## Roster
The canonical list lives in `team.json` (`operatingContract.members`). Roles, in summary:
- `programmatic-qa-runner` — runs GCT readiness reviews; produces `qa-run/*` knowledge and `preemptive-qa-backlog` decisions.
- `quality-auditor` — applies the seven-lens audit rotation; produces `quality-audit/*` knowledge and `quality-audit-backlog` decisions.
- `bug-investigator` — drains `bug-inbox/*` (universal-source); applies investigation techniques; produces `bug-investigation-report/*` audit log and `bug-resolution-proposal` decisions.
- `qa-contrarian` — reads peer outputs; produces `challenge-report/*` citing specific failure modes from the technique registries.

## Knowledge topic table

| Topic prefix | Owner | Producer | Retention |
|---|---|---|---|
| `qa-run/<scenario-id>` | programmatic-qa-runner | self | append-only |
| `reviewed-scenario/<scenario-id>` | programmatic-qa-runner | self | append-only |
| `dependency-wiring` | programmatic-qa-runner | self | append-only |
| `quality-audit/<scenario-id>/<skill-id>` | quality-auditor | self | append-only |
| `bug-inbox/<signal-type>/<slug>` | bug-investigator | **any team via `report-bug` skill** (universal-source intake) | drained-on-close |
| `bug-investigation-report/<slug>` | bug-investigator | self | append-only |
| `challenge-report/<slug>` | qa-contrarian | self | append-only |

## Decision contexts
- `preemptive-qa-backlog` — owner: programmatic-qa-runner. GCT-driven QA findings → Swarm Manager fix/chore/execute backlog items.
- `quality-audit-backlog` — owner: quality-auditor. Judgment-based structural audit findings → Swarm Manager execute backlog items.
- `bug-resolution-proposal` — owner: bug-investigator. Cross-cutting fixes that require operator approval (rename a CLI verb, refactor a confusing skill section, canonicalize a drifted data-shape, etc.).

## Universal-source intake pattern
`bug-inbox/*` declares `source_team: "*"` in the bug-investigator's `topics.json`. This is a first-class semantic meaning *any team's members may write*. The producer-side anchor is the `report-bug` writer skill — declared in the bug-investigator's `external_producers` so the validator's `wildcard_source_misuse` rule stays quiet. The investigator validates the producer's signal-type assignment as the first sub-step of investigation; there is no separate classifier skill (deterministic-prefix routing).

## Team-Specific Principles
- **Findings become artifacts, not direct edits.** Backlog items, audit-log entries, investigation logs, challenge notes — all evidence-rich and actionable by future agents who don't have this team's context.
- **Behavior-oriented evidence over generic audit language.** "The X output produced Y when given Z, contradicting the schema documented at W" beats "this could be improved."
- **Investigation rigor.** One technique applied per entry per heartbeat; if a technique stalls, record findings honestly and move on. A bug-investigation that admits "blocked, capability-gap filed" is more honest than one that guesses at a cause.
- **Contrarian discipline.** Every challenge cites a specific failure mode from a registered technique's PoR doc. Quiet heartbeats are valid; manufactured challenges are forbidden.
- **Right inbox for the observation.** Bugs go to `bug-inbox/*` via the `report-bug` skill — even when observed during readiness review or quality audit. Each member uses the inbox shape that fits its observation.
- **Doc + paired skill discipline.** Every technique has both a strategic-canon doc (in `docs/scenario-qa/{investigation,audit,readiness}-techniques/`) and an executable skill. Neither half is optional.
