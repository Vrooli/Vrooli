# Scenario-QA Team Plan-of-Record + Bug-Inbox: Plan of Action

Status: draft. Authored 2026-05-03. Will move out of `drafts/` once execution begins.

This is the proper long-term implementation: a full plan-of-record for the `scenario-qa` team (currently has none), parallel technique registries for each of the team's work types (investigation, audit, readiness — paired-doc-and-skill, mirroring marketing's `post-techniques/` pattern), two new members (`bug-investigator` draining a cross-team `bug-inbox/*`, `qa-contrarian` matching the contrarian-per-team pattern), full updates to existing members so they cite the new registries, and the schema upgrade required to make universal-source intakes a first-class concept.

After this lands, scenario-qa has the same operating standard as `marketing-crew` and `monetization`: full strategic canon, paired technique registries, complete member roster including a contrarian, and an enabled live drain.

## 1. Required Reading

```bash
prompt-manager skill read decision-boundary-extraction seam-discovery-and-enforcement boundary-of-responsibility-enforcement scientific-debugging team-shared-docs-pattern
```

Plus PoR files governing the substrate being changed:

- `docs/agent-system/INTAKE_PIPELINE.md` — two routing modes; cross-team schema ownership
- `docs/agent-system/TOPICS.md` — registry; notebook-vs-typed-inbox; scenario-qa observations
- `docs/agent-system/TOPICS_SCHEMA.md` — schema; cross-team flow
- `docs/agent-system/PRIMITIVES.md`
- `docs/agent-system/LAYERS.md`
- `docs/agent-system/TEAM_DOCS_PATTERNS.md` — paired-doc-and-skill discipline; per-entity files vs monolithic
- `docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md` — 9-layer member capability model; smells (skillless canon, mega-skill pressure, etc.)
- `docs/agent-system/PROMOTION_LADDER.md` — how techniques mature
- `docs/marketing/post-techniques/README.md` — gold-standard reference for the technique-registry pattern this plan replicates
- `docs/marketing/post-types/README.md` — doc + paired skill rule (mandatory hard rule, not recommendation)
- `scenarios/prompt-manager/api/memberflow/schema.go` and `validation.go`
- `scenarios/prompt-manager/api/heartbeat/inbox_flow.go` and `prompt_builder.go`
- `scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/{screaming-architecture-audit,boundary-of-responsibility-enforcement,seam-discovery-and-enforcement,invariant-discovery-and-enforcement,cognitive-load-reduction,decision-boundary-extraction,code-cleanup}/SKILL.md`
- `scenarios/prompt-manager/store/agents/quality-auditor/` (agent template reference)
- `scenarios/prompt-manager/store/agents/marketing-contrarian/` (contrarian template reference)
- `scenarios/prompt-manager/store/teams/scenario-qa/team.json` (note `enabled: false`, narrow mission)
- `scenarios/prompt-manager/store/teams/scenario-qa/members/{programmatic-qa-runner,quality-auditor}/` (existing member files)

## 2. Context

scenario-qa is currently the runt of the team set. Compared with peer teams (`marketing-crew`, `monetization`, `meta-optimization`, `infra-health`):

1. **No plan-of-record folder.** Other domain teams own a `docs/<domain>/` folder of operator-curated canon; scenario-qa does not. Every audit and bug investigation reinvents the methodology. The 7 audit lenses in `quality-auditor`'s `skillRotation` are skill-only — no paired strategic-canon docs explaining when each applies, when it backfires, what failure mode the contrarian should watch for. This is the `skillless canon` smell from `TEAM_MEMBER_ARCHITECTURE.md`.
2. **No technique registries.** Marketing organizes its cross-cutting voice/structure techniques in `post-techniques/` and its per-output strategic canon in `post-types/`. scenario-qa has no equivalent registries — three distinct work types (investigation, audit, readiness-check) all live as ungrouped skills with no canonical home and no documented graduation path for new techniques.
3. **No taxonomies, no inboxes.** Both members are pure proactive producers (`intake[]: []`). External signals — bugs discovered in any other team's work, operator-fed alpha, cross-team handoff — have no structured channel into scenario-qa.
4. **No contrarian member.** Every other team has one (`marketing-contrarian`, `monetization/contrarian`, `meta-contrarian`, `infra-contrarian`). scenario-qa is the only outlier; QA outcomes go un-challenged by structure.
5. **Team disabled.** `enabled: false` in `team.json` — even the existing two members aren't active.
6. **Narrow mission.** Today: "Ensure scenario quality through code auditing, test coverage analysis, and documentation verification." The mission doesn't anticipate bug investigation, contrarian challenge, or anything signal-shaped.

A bug-inbox plan that adds `bug-investigator` alone closes none of those gaps — it would compound them by adding a new member to a team that has no operating standard. This plan addresses all six together. They share the same execution boundary (one team's substrate), and the technique-registry pattern is the only honest way to organize the team's growing skill surface (seven audit lenses already exist, more investigation techniques will graduate over time).

A schema gap also surfaced during planning: `bug-inbox/*` will receive writes from every team's members — a topology not currently supported by the validator. A real schema upgrade is required (`source_team: "*"` as a first-class declaration).

## 3. Goals

1. **scenario-qa plan-of-record.** A new `docs/scenario-qa/` folder containing the team's full strategic canon: README, three technique registries (investigation, audit, readiness), bug-report taxonomy. Operator-curated like every other team PoR.
2. **Investigation-techniques registry.** `docs/scenario-qa/investigation-techniques/` mirrors `docs/marketing/post-techniques/` — one paired doc-and-skill per technique. `scientific-debugging` is the first entry, paired with the existing skill.
3. **Audit-techniques registry.** `docs/scenario-qa/audit-techniques/` — seven paired PoR docs for `quality-auditor`'s seven existing audit lenses. Closes the `skillless canon` smell across the entire audit rotation in a single pass.
4. **Readiness-checks registry (skeleton).** `docs/scenario-qa/readiness-checks/README.md` — stub now; full population deferred until GCT dimensions stabilize. Documented as future PoR work.
5. **Universal bug capture.** Any agent on any team records a bug observation in one skill invocation, regardless of which scenario, skill, or component is broken.
6. **Specialist triage with extensible methodology.** A new `bug-investigator` member on scenario-qa drains the inbox using the investigation-techniques registry. New techniques graduate via `meta-self-improvement` decisions.
7. **Contrarian challenge.** A new `qa-contrarian` matches the cross-team pattern; reads peer-team decisions; writes `challenge-report/*` that surfaces gaps in QA reasoning, false-positive findings, and over-reach.
8. **Existing members updated to the new standard.** `quality-auditor` and `programmatic-qa-runner` get RESPONSIBILITIES.md updates that reference the technique registries; their `topics.json` files reviewed against the 9-layer model.
9. **Schema upgrade for universal-source intakes.** `intake[].source_team: "*"` becomes a first-class declaration meaning "any team's members may write." Validator handles correctly. Documented in `TOPICS_SCHEMA.md`.
10. **bug-report taxonomy.** Six signal types, schemas for both bug-report and bug-investigation entries, action-selection rules, evidence rules. JSON sidecar + markdown PoR pair.
11. **Storage Map integration.** Every agent's heartbeat points to `report-bug` as the canonical write-side skill for bug observations.
12. **Updated team config and live triage.** Mission rewritten to encompass new scope; member roster expanded; decision contexts added; team enabled.
13. **Zero net behavior change for non-scenario-qa members.** Existing inboxes, taxonomies, classifiers, members, and decisions on other teams are not affected.

## 4. Non-Goals

- **Other observation inboxes (capability-gap, friction).** Already handled by existing decision contexts and the Storage Map fan-out. Out of scope.
- **`qa-inbox/*` or `audit-inbox/*` for existing members.** No producer is identified; would create `orphan_input`. Documented as future PoR work in the team README; not added in this plan.
- **Authoring beyond the scientific-debugging investigation technique.** Future investigation techniques (bisect-debugging, minimal-reproduction, differential-trace, comparative-environments, 5-whys, fishbone) follow the standard `meta-self-improvement` decision flow.
- **Authoring readiness-checks entries.** Skeleton README only; concrete entries arrive when GCT dimensions stabilize.
- **Migrating existing notebook entries that are bugs.** Bugs already in notebooks stay there; the curator promotes/retires per usual notebook-debt rules. New bugs go to bug-inbox.
- **Filling out additional scenario-qa strategic canon** beyond the README and three registries (e.g., quality principles, audit-dimension rubric, scenario-classification heuristics). Flagged as future PoR work in the README; not authored here.
- **Splitting bug-investigator into specialist sub-investigators.** Single member; if volume requires it later, splitting is a follow-up plan.
- **Renaming the team itself.** Team stays `scenario-qa`. Folder is `docs/scenario-qa/`.
- **Authoring readiness-check or audit-technique paired skills that don't already exist.** All seven audit-technique skills already exist; this plan only adds the paired PoR docs. New skills are authored later under the standard graduation flow.

## 5. Greenfield Statement

**This is greenfield work, end-to-end.** New team PoR folder, three new technique registries, new taxonomy, two new agent identities, two new member bindings, new writer skill, new schema field, new validator behavior, rewritten team mission, paired strategic-canon docs for seven existing audit skills. No backwards-compatibility shims, no aliases, no v1/v2 deferrals — every architectural decision is settled in §6 and implemented in §7. After this lands, scenario-qa has a complete operating surface (team PoR, technique registries, four-member roster including contrarian, bug-inbox drain duty, enabled team) by the same standard the marketing and monetization teams meet today.

## 6. Architecture Summary

### 6.1 Layer table

| Concern | Home |
|---|---|
| Team mission, scope, principles, decision contexts, knowledge topics | `docs/scenario-qa/README.md` (new — strategic canon overview) + `store/teams/scenario-qa/shared/TEAM.md` (charter) |
| Bug-report capture procedure | `report-bug` skill (writer skill, opt-in via Storage Map pointer) |
| Bug-report classification | Deterministic-prefix routing — producer picks signal-type; investigator validates as the first step of investigation. No separate classifier skill. (See §6.11 for rationale.) |
| Investigation methodology canon | `docs/scenario-qa/investigation-techniques/<slug>.md` (one PoR doc per technique) |
| Investigation methodology procedure | `scenarios/prompt-manager/store/skills/packs/core/<technique-slug>/SKILL.md` (one skill per technique; doc + paired skill discipline) |
| Audit-technique canon | `docs/scenario-qa/audit-techniques/<slug>.md` (one PoR doc per existing audit-lens skill) |
| Audit-technique procedure | existing skill at `scenarios/prompt-manager/store/skills/packs/core/<slug>/SKILL.md` (already authored; cross-linked in this plan) |
| Readiness-check canon | `docs/scenario-qa/readiness-checks/<slug>.md` (skeleton — stub README only in this plan; entries deferred) |
| Drain procedure | `scenario-qa/bug-investigator` member's heartbeat — Inbox Flow section generated from `topics.json` |
| Triage outcomes | bug-report taxonomy's `actionSelection` (drop / observe / file-backlog / file-decision / route-to-another-topic / capability-gap) |
| Universal-source declaration | `intake[].source_team: "*"` in `topics.json` (new schema semantics) |
| Bug investigation audit log | `bug-investigation-report/<slug>` knowledge entries (one per closed bug — root cause, technique applied, action taken) |
| Cross-team contrarian challenge | `qa-contrarian` — writes `challenge-report/*` per the cross-team contrarian pattern |

### 6.2 `docs/scenario-qa/` folder structure

```
docs/scenario-qa/
├── README.md                              # team PoR overview (strategic canon)
├── investigation-techniques/
│   ├── README.md                          # registry; lifecycle rules; doc + paired skill mandate
│   └── scientific-debugging.md            # paired with existing scientific-debugging skill
├── audit-techniques/
│   ├── README.md                          # registry; same lifecycle/mandate as investigation
│   ├── screaming-architecture-audit.md    # paired with existing skill of same slug
│   ├── boundary-of-responsibility-enforcement.md
│   ├── seam-discovery-and-enforcement.md
│   ├── invariant-discovery-and-enforcement.md
│   ├── cognitive-load-reduction.md
│   ├── decision-boundary-extraction.md
│   └── code-cleanup.md
├── readiness-checks/
│   └── README.md                          # stub; future PoR work flagged
├── bug-report-taxonomy.json               # JSON sidecar
└── BUG_REPORT_TAXONOMY.md                 # human-readable view
```

The slug-of-doc must match the slug-of-skill (canon coherence test enforces this). Doc filenames carry the skill suffix when one exists (e.g., `screaming-architecture-audit.md`, not `screaming-architecture.md`) so the pairing is unambiguous.

### 6.3 Team mission rewrite

Current `team.json` mission:

> Ensure scenario quality through comprehensive code auditing, test coverage analysis, and documentation verification.

New:

> Ensure scenario quality through deep architectural audits, programmatic readiness reviews, root-cause bug investigation, and contrarian challenge of QA outcomes — producing evidence-rich backlog items, audit logs, investigation logs, and challenge notes that downstream agents act on.

`shared/TEAM.md` mission line and team-specific principles updated in parallel; principles list expanded to mention investigation rigor (one technique applied, one outcome) and contrarian discipline.

### 6.4 Bug-inbox topic shape

```
bug-inbox/<signal-type>/<short-slug>
```

Signal types (deterministic — producer picks at write time, investigator validates):

- `code-defect` — broken code or scenario behavior
- `regression` — used to work, now doesn't
- `prompt-confusion` — agent misled by ambiguous or contradictory prompt content
- `data-shape-mismatch` — observed payload didn't match documented schema
- `unexpected-error` — error not documented anywhere
- `unknown` — type unclear; investigator classifies during triage

### 6.5 Front-matter schemas

**bug-report** (entry written by reporter):

```yaml
severity: blocker | major | minor
reporter: <agent-id>
reporter_team: <team-id>
observed_at: <YYYY-MM-DD>
context:
  scenario: <scenario-id-or-null>
  skill: <skill-id-or-null>
  member: <member-id-or-null>
  command: <CLI-or-null>
repro:
  - <step 1>
  - <step 2>
expected: <one-line>
actual: <one-line>
description: |
  <free-form notes>
honesty_flags: [<flags>]
```

**bug-investigation** (entry written by investigator on close):

```yaml
bug_id: <source-knw-id>
investigator: <agent-id>
technique: <technique-skill-id>
outcome: drop | observe | file-backlog | file-decision | route-to-another-topic | capability-gap
root_cause: <text-or-null>
fix_target: <text-or-null>
closed_at: <YYYY-MM-DD>
```

Body required sections for `bug-investigation`: `Findings`, `Action taken`.

### 6.6 Schema change: `source_team: "*"`

`IntakeEntry.source_team` accepts:

- `null` — same-team source or external producer (existing)
- `<team-id>` — specific cross-team source (existing)
- `"*"` (new) — universal: any team's members may write. Validator skips `orphan_input` for the entry.

Documented in `TOPICS_SCHEMA.md` § Cross-team flow + new § Universal-source intakes subsection. New validator rule `wildcard_source_misuse` (warning) catches `source_team == "*"` paired with empty `external_producers[]` — i.e., "I made it universal but forgot to document the producer-side anchor."

### 6.7 Investigation-techniques registry

Mirrors `docs/marketing/post-techniques/` exactly:

- **Doc + paired skill discipline (mandatory).** Every technique ships as a strategic-canon doc *and* an executable skill. A doc with no skill is a stale shrine; a skill with no doc is brittle. Both halves required, no exceptions.
- **One technique, one folder location.** Cross-cutting techniques (those a `bug-investigator` may apply across multiple signal types) get a single canonical home.
- **Lifecycle: v0 (doc-only stub) → v1 (doc + skill, active).** v0 means strategic canon is documented but the technique is not yet active. Activation requires (1) skill is authored, (2) skill cites the technique doc as required reading, (3) doc Status line bumped to v1, (4) `bug-investigator/RESPONSIBILITIES.md` references the skill in its Available Skills table.
- **Compression operates per-skill.** Each technique's skill compresses independently as Vrooli's substrate (CLIs, debug tooling) absorbs more of the work. A unified `bug-investigate` mega-skill that branches on technique would compress worse — same argument the marketing team uses for one skill per post type.
- **Adding a technique.** New techniques enter via `meta-self-improvement` decision proposing the addition. The bug-investigator surfaces graduation candidates from observed patterns in `bug-investigation-report/<slug>` audit entries.

`scientific-debugging.md` is the first entry. The existing `scientific-debugging` skill is updated to add `docs/scenario-qa/investigation-techniques/scientific-debugging.md` to its required-reading list.

### 6.8 Audit-techniques registry

Same pattern, applied to `quality-auditor`'s seven existing skills. Each PoR doc answers:

- **What is this audit lens?** One-paragraph definition in plain English.
- **When does it apply?** Which scenarios benefit most (e.g., monolithic packages, deep call chains, cross-cutting concerns).
- **When does it backfire?** Failure modes (e.g., applied prematurely to a feature that's still being scoped; over-applied and producing noise).
- **What does the qa-contrarian watch for?** Specific failure signals (e.g., audit findings that propose churn without business value, audits that ignore cross-cutting concerns the lens isn't designed to surface).
- **Cites the existing skill** as the executable spec.

Seven entries authored at landing time:

| Doc slug | Paired skill |
|---|---|
| `screaming-architecture-audit` | existing `screaming-architecture-audit` skill |
| `boundary-of-responsibility-enforcement` | existing skill of same slug |
| `seam-discovery-and-enforcement` | existing skill of same slug |
| `invariant-discovery-and-enforcement` | existing skill of same slug |
| `cognitive-load-reduction` | existing skill of same slug |
| `decision-boundary-extraction` | existing skill of same slug |
| `code-cleanup` | existing skill of same slug |

Each existing skill's required-reading list is updated to cite its new PoR doc. `quality-auditor`'s `RESPONSIBILITIES.md` gains an Available Skills table listing all seven, mirroring the marketing pattern.

This closes the `skillless canon` smell across the audit rotation in a single pass. New audit techniques (e.g., performance-audit, security-audit, deprecation-audit) graduate into the registry via `meta-self-improvement` decisions.

### 6.9 Readiness-checks registry (stub)

`docs/scenario-qa/readiness-checks/README.md` is a stub: explains the registry's purpose (paired strategic-canon docs for `programmatic-qa-runner`'s individual readiness dimensions and check categories), declares the doc + paired skill discipline applies here too, lists "Files in this folder: none yet — populated as GCT dimensions stabilize and individual readiness checks become candidate skills."

No concrete entries authored in this plan. Documented as future PoR work in the README and in scenario-qa's main README.md.

This is intentional asymmetry with audit-techniques: the audit lenses are seven mature skills already in active rotation, so pairing them is mechanical. GCT readiness dimensions are external (driven by GCT itself, not by Vrooli skills) and the right registry shape isn't yet known. Stubbing now reserves the canonical home; entries follow when the shape is clearer.

### 6.10 Doc + paired skill discipline (cross-cutting rule)

All three registries follow the same mandatory rule from `docs/marketing/post-types/README.md`:

> Every entry ships as `doc + paired skill`. This is a hard rule, not a recommendation. Neither half is optional, and neither half replaces the other. The doc holds *reasoning*; the skill holds *procedure*. A doc with no skill is a stale shrine. A skill with no doc is brittle.

Enforced by a canon coherence test (§9) that checks every `<registry>/<slug>.md` (excluding README.md) has a matching `store/skills/packs/core/<slug>/SKILL.md`, and vice versa for skills tagged with the registry's tag.

### 6.11 No classifier skill — deterministic-prefix routing (settled)

Bug-inbox uses deterministic-prefix routing without a classifier skill. The producer assigns the signal-type at write time (via the `report-bug` skill, which prompts for it from a fixed list); the bug-investigator validates the assignment as the first step of investigation.

This matches the notebook-debt pattern (curator does classification as part of curation), not the marketing-research pattern (separate classifier skill). The reason: **bug investigation is fundamentally one workflow that includes classification as a sub-step**. Separating classification into its own skill creates a hop without value because the investigator must read the entry anyway to start investigation. The classifier's hypothetical role (assign signal-type, evidence-strength, honesty-flags) is identical to the first three steps of investigation.

This is not a deferral. Adding a classifier later would be architectural drift, not an upgrade.

### 6.12 `bug-investigator` member

Single-team binding to scenario-qa.

`topics.json`:

```jsonc
{
  "intake": [
    {
      "prefix": "bug-inbox/*",
      "taxonomy": "bug-report",
      "source_team": "*"
    }
  ],
  "output": [
    { "prefix": "bug-investigation-report/*", "destination_kind": "knowledge", "destination_team": null }
  ],
  "decisions_owned": ["bug-resolution-proposal"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["any-team-member-via-report-bug-skill"]
}
```

`bug-investigation-report/<slug>` is the durable audit log — one entry per closed bug, capturing the technique applied, root cause, and action taken. Used to surface graduation candidates for new investigation techniques and to detect repeat bugs.

`bug-resolution-proposal` decision context covers cross-cutting fixes that require operator approval (e.g., "rename this CLI verb because three bugs trace to its ambiguous name").

Member files created from the quality-auditor template:

- `store/agents/bug-investigator/SOUL.md` — identity (specialist scientific-debugger; calm, methodical, no speculation; honest when investigation is blocked)
- `store/agents/bug-investigator/AGENTS.md` — workflow contract (load technique skill from registry, drain top of inbox, investigate, classify outcome, write `bug-investigation-report/<slug>` log, hand off via swarm-manager / decision / route)
- `store/agents/bug-investigator/TOOLS.md` — bindings: every technique skill from the investigation-techniques registry (currently `scientific-debugging`), swarm-manager CLI, prompt-manager team knowledge-list/update/delete (no add — bug-investigator never writes to its own inbox)
- `store/agents/bug-investigator/agent.json` — metadata
- `store/teams/scenario-qa/members/bug-investigator/topics.json` — above
- `store/teams/scenario-qa/members/bug-investigator/HEARTBEAT.md` — heartbeat task: drain top of inbox by severity, pick technique from registry, investigate, write audit-log entry, take action
- `store/teams/scenario-qa/members/bug-investigator/RESPONSIBILITIES.md` — drain duty + investigation rigor + escalation rules + Available Skills table listing every technique
- `store/teams/scenario-qa/members/bug-investigator/last-handoff.md` — initialized empty

### 6.13 `qa-contrarian` member

Matches the cross-team contrarian pattern (`marketing-contrarian`, `monetization/contrarian`, `meta-contrarian`, `infra-contrarian`). Proactive — reads peer-team decisions and member outputs; writes `challenge-report/*` to scenario-qa's knowledge store.

`topics.json`:

```jsonc
{
  "intake": [],
  "output": [
    { "prefix": "challenge-report/*", "destination_kind": "knowledge", "destination_team": null }
  ],
  "decisions_owned": [],
  "decisions_consumed": [],
  "raises_capability_gaps": false,
  "external_producers": []
}
```

The qa-contrarian's job is to challenge:
- Bug-investigation outcomes (was the root cause actually identified, or was the investigator stopped at first plausible explanation?)
- Audit findings (does the proposed change actually deliver value, or is it churn-for-churn's-sake?)
- Readiness-review backlog items (is the GCT failure a real defect or a documentation gap?)
- Cross-team handoffs from scenario-qa (does the backlog item have enough evidence to act without scenario-qa's context?)

Member files created from the marketing-contrarian template:

- `store/agents/qa-contrarian/SOUL.md` — identity (skeptical, first-principles, allergic to consensus, honest when no challenge is warranted)
- `store/agents/qa-contrarian/AGENTS.md` — workflow contract (read peer decisions on cadence, surface specific failure modes, write challenge-report/<slug>)
- `store/agents/qa-contrarian/TOOLS.md` — bindings: prompt-manager team decision-list/knowledge-list, swarm-manager CLI for backlog-item review
- `store/agents/qa-contrarian/agent.json` — metadata
- `store/teams/scenario-qa/members/qa-contrarian/topics.json` — above
- `store/teams/scenario-qa/members/qa-contrarian/HEARTBEAT.md` — heartbeat task: scan recent bug-investigation entries + peer-team decisions; write at most N challenge-notes per heartbeat
- `store/teams/scenario-qa/members/qa-contrarian/RESPONSIBILITIES.md` — challenge discipline; signal-vs-noise rules; what NOT to challenge
- `store/teams/scenario-qa/members/qa-contrarian/last-handoff.md` — initialized empty

`challenge-report/*` orphan-output applies (per TOPICS.md known inconsistency #3 — the cross-team contrarian-drain question is workshop-pending and out of scope).

### 6.14 `quality-auditor` updates

Existing member; updated to follow the new standard.

- `store/teams/scenario-qa/members/quality-auditor/RESPONSIBILITIES.md`:
  - Add "Available Skills" table with all 7 audit-technique skills, each with a one-line "when to apply" hook.
  - Add cross-link to `docs/scenario-qa/audit-techniques/README.md` as the canon hub for the rotation.
  - Add cross-link to `docs/scenario-qa/README.md` for team-level context.
- `store/teams/scenario-qa/members/quality-auditor/HEARTBEAT.md`:
  - Add cross-link to `docs/scenario-qa/audit-techniques/<current-rotation-slug>.md` so the auditor reads the strategic-canon doc before applying the lens.
- `store/teams/scenario-qa/members/quality-auditor/topics.json`:
  - Reviewed against 9-layer model. Currently fine: proactive output to `quality-audit/*`, owns `deep-audit-backlog`. No structural changes needed.

### 6.15 `programmatic-qa-runner` updates

Existing member; updated to follow the new standard.

- `store/teams/scenario-qa/members/programmatic-qa-runner/RESPONSIBILITIES.md`:
  - Add cross-link to `docs/scenario-qa/readiness-checks/README.md` (stub — flagged as future PoR work).
  - Add cross-link to `docs/scenario-qa/README.md` for team-level context.
- `store/teams/scenario-qa/members/programmatic-qa-runner/HEARTBEAT.md`:
  - Cross-link to readiness-checks README (stub).
- `store/teams/scenario-qa/members/programmatic-qa-runner/topics.json`:
  - Reviewed against 9-layer model. Currently fine: proactive output to `qa-run/*`, owns `preemptive-qa-backlog`. No structural changes needed.

### 6.16 `report-bug` skill

Universal writer skill any agent can invoke. Lives in the `core` skill pack; discovered via `prompt-manager skill discover --query="bug"` and pointed-to from the heartbeat Storage Map.

- `store/skills/packs/core/report-bug/skill.json` — id `report-bug`, modes `["tools"]`, tags `["observability", "bug-report"]`
- `store/skills/packs/core/report-bug/SKILL.md`:
  - Required reading: `docs/scenario-qa/BUG_REPORT_TAXONOMY.md`
  - Inputs: signal-type (from fixed list), severity, repro, expected, actual, context, description
  - Procedure: validate inputs against schema; generate kebab-case slug; construct `bug-inbox/<signal-type>/<slug>`; format front-matter + body; invoke `prompt-manager team knowledge-add scenario-qa --by=<reporter-agent-id> --topic="..." --content="..."`
  - Output: confirms entry id

The skill is destination-coupled by design (writer skills always are; the `non_portable_classifier` rule applies only to classifier skills).

**Auto-loading vs. on-demand.** The skill is **on-demand** — discovered via the Storage Map pointer, loaded when an agent decides to report a bug. Auto-loading into every heartbeat would bloat agent context with rarely-used procedure. The Storage Map pointer is the discoverability anchor; the skill carries the procedure.

### 6.17 `bug-report` taxonomy

`docs/scenario-qa/bug-report-taxonomy.json`:

```jsonc
{
  "schemaVersion": 1,
  "id": "bug-report",
  "displayName": "Bug Report Triage Taxonomy",
  "owner_team": "scenario-qa",
  "porPath": "docs/scenario-qa/BUG_REPORT_TAXONOMY.md",
  "signalTypes": [
    { "id": "code-defect",         "definition": "Broken code or scenario behavior",                 "defaultMethod": "scientific-debugging" },
    { "id": "regression",          "definition": "Worked previously; now broken",                    "defaultMethod": "scientific-debugging" },
    { "id": "prompt-confusion",    "definition": "Agent misled by ambiguous/contradictory prompt",   "defaultMethod": "scientific-debugging" },
    { "id": "data-shape-mismatch", "definition": "Payload didn't match documented schema",           "defaultMethod": "scientific-debugging" },
    { "id": "unexpected-error",    "definition": "Error not documented anywhere",                    "defaultMethod": "scientific-debugging" },
    { "id": "unknown",             "definition": "Type unclear; investigator must classify",         "defaultMethod": "scientific-debugging" }
  ],
  "evidenceRules": [
    "A repro is mandatory; if absent, investigator first tries to reproduce, else routes to file-decision asking for repro.",
    "Severity is the producer's claim; the investigator may overrule based on actual scope of impact.",
    "Honesty flags must include `repro-not-attempted` if the producer didn't try to reproduce.",
    "Investigation outcomes must reference the technique applied (e.g., `scientific-debugging`) so the audit log can drive technique-graduation decisions."
  ],
  "actionSelection": {
    "drop": "Cannot reproduce + no clear scope; weak one-off.",
    "observe": "Confirmed bug; route findings to `bug-investigation-report/<slug>` and continue (no fix required this heartbeat).",
    "file-backlog": "Reproducible; investigator hands off swarm-manager backlog item with full evidence.",
    "file-decision": "Cross-cutting; investigator raises `bug-resolution-proposal` for operator review.",
    "route-to-another-topic": "Misclassified — actually a documentation gap, capability-gap, or skill-issue. Retag/rewrite to the appropriate inbox or file the appropriate decision.",
    "capability-gap": "Repro requires missing tool/scenario/CLI; raise `capability-gap` decision and leave the inbox entry until the gap is closed."
  },
  "schemas": {
    "bug-report": {
      "frontMatter": {
        "severity": "<blocker|major|minor>",
        "reporter": "<agent-id>",
        "reporter_team": "<team-id>",
        "observed_at": "<YYYY-MM-DD>",
        "context": "<context object>",
        "repro": "[<steps>]",
        "expected": "<text>",
        "actual": "<text>",
        "description": "<text>",
        "honesty_flags": "[<flags>]"
      },
      "bodyRequiredSections": []
    },
    "bug-investigation": {
      "frontMatter": {
        "bug_id": "<source-knw-id>",
        "investigator": "<agent-id>",
        "technique": "<technique-skill-id>",
        "outcome": "<drop|observe|file-backlog|file-decision|route-to-another-topic|capability-gap>",
        "root_cause": "<text-or-null>",
        "fix_target": "<text-or-null>",
        "closed_at": "<YYYY-MM-DD>"
      },
      "bodyRequiredSections": ["Findings", "Action taken"]
    }
  },
  "honestyFlags": ["repro-not-attempted", "speculative-cause", "minimal-context", "ai-generated-summary"]
}
```

`docs/scenario-qa/BUG_REPORT_TAXONOMY.md` — human-readable view; cites the JSON sidecar; explains each signal type with examples; documents the operator's view of bug-investigation outputs.

## 7. Phased Implementation Steps

### Phase A — Schema and validation

**Goal:** `topics.json` supports `source_team: "*"`; validator handles correctly. No agent-visible change yet.

**Files (modified):**
- `scenarios/prompt-manager/api/memberflow/schema.go` — `IntakeEntry.SourceTeam` semantics extended; `"*"` literal accepted
- `scenarios/prompt-manager/api/memberflow/validation.go`:
  - `orphan_input` rule: when `source_team == "*"`, skip producer-existence check
  - New rule `wildcard_source_misuse` (warning): `source_team == "*"` AND `external_producers` does not name any anchor

**Tests:** `schema_test.go` (parsing); `validation_test.go` (positive and negative cases for both rule changes).

### Phase B — Author scenario-qa team plan-of-record

**Files (new):**
- `docs/scenario-qa/README.md` — team PoR overview, structured as the marketing/monetization READMEs are:
  - Mission (matches §6.3 rewrite; cross-link to `store/teams/scenario-qa/shared/TEAM.md`)
  - Scope and boundaries
  - Folder map (links to investigation-techniques, audit-techniques, readiness-checks, BUG_REPORT_TAXONOMY)
  - Decision contexts owned (`bug-resolution-proposal`, `deep-audit-backlog`, `preemptive-qa-backlog`)
  - Member roster (`programmatic-qa-runner`, `quality-auditor`, `bug-investigator`, `qa-contrarian`)
  - Cross-team relationships (downstream: swarm-manager backlog, meta-optimization decisions; upstream: producers via `bug-inbox/*`)
  - "Future PoR work" section explicitly listing what's not yet authored: quality principles, scenario-classification heuristics, `qa-inbox/*`/`audit-inbox/*` operator-fed inboxes if a producer emerges, full readiness-checks registry once GCT dimensions stabilize.

### Phase C — Author investigation-techniques registry

**Files (new):**
- `docs/scenario-qa/investigation-techniques/README.md`:
  - Purpose: strategic canon for techniques the bug-investigator applies
  - Doc + paired skill discipline (mandatory hard rule, mirroring `docs/marketing/post-types/README.md`)
  - Lifecycle: v0 → v1 with the four activation requirements
  - Compression-per-skill rationale (cross-link to `PROMOTION_LADDER.md`)
  - Files-in-this-folder table (currently one entry: scientific-debugging)
  - Adding a technique: process (file `meta-self-improvement` decision, operator approves, paired doc + skill authored, registry table updated)
- `docs/scenario-qa/investigation-techniques/scientific-debugging.md`:
  - Status: v1 (paired with existing `scientific-debugging` skill)
  - Strategic canon: when this technique applies, when it backfires, what failure modes the qa-contrarian watches for
  - Cites the existing skill as the executable spec
  - Cites `docs/agent-system/SKILL_AUTHORING.md`

**Files (modified):**
- `scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md` — add `docs/scenario-qa/investigation-techniques/scientific-debugging.md` to required reading; cross-link the strategic canon

### Phase D — Author audit-techniques registry

**Files (new):**
- `docs/scenario-qa/audit-techniques/README.md`:
  - Purpose: strategic canon for `quality-auditor`'s audit lenses
  - Same doc + paired skill discipline
  - Same lifecycle rules as investigation-techniques
  - Files-in-this-folder table with all seven entries
  - Adding a technique: same process as investigation-techniques
- `docs/scenario-qa/audit-techniques/screaming-architecture-audit.md`
- `docs/scenario-qa/audit-techniques/boundary-of-responsibility-enforcement.md`
- `docs/scenario-qa/audit-techniques/seam-discovery-and-enforcement.md`
- `docs/scenario-qa/audit-techniques/invariant-discovery-and-enforcement.md`
- `docs/scenario-qa/audit-techniques/cognitive-load-reduction.md`
- `docs/scenario-qa/audit-techniques/decision-boundary-extraction.md`
- `docs/scenario-qa/audit-techniques/code-cleanup.md`

Each PoR doc follows the structure in §6.8: definition, when-applies, when-backfires, qa-contrarian-failure-modes, citation of existing skill.

**Files (modified):**
Each of the seven existing skills' `SKILL.md` is updated to add its paired PoR doc to required reading:
- `scenarios/prompt-manager/store/skills/packs/core/screaming-architecture-audit/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/boundary-of-responsibility-enforcement/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/seam-discovery-and-enforcement/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/invariant-discovery-and-enforcement/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/cognitive-load-reduction/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/decision-boundary-extraction/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/code-cleanup/SKILL.md`

Each of these skills also gains a `tags: ["audit-technique"]` entry in `skill.json` for the canon coherence test pairing check.

### Phase E — Author readiness-checks registry (stub)

**Files (new):**
- `docs/scenario-qa/readiness-checks/README.md` — stub per §6.9: purpose, doc + paired skill discipline applies here too, "Files in this folder: none yet — populated as GCT dimensions stabilize." Future PoR work flagged.

### Phase F — Author bug-report taxonomy

**Files (new):**
- `docs/scenario-qa/bug-report-taxonomy.json` (per §6.17)
- `docs/scenario-qa/BUG_REPORT_TAXONOMY.md` — human-readable view; cite JSON sidecar; explain signal types with examples

**Files (modified):**
- `docs/agent-system/README.md` — add `bug-report` row to § Active taxonomies registry (handled here so the registry is current)

**Tests:** `taxonomy_authoring_test.go` extension — taxonomy loads; both schema ids (`bug-report`, `bug-investigation`) resolve; `defaultMethod: "scientific-debugging"` resolves to a registered skill.

### Phase G — Author `report-bug` skill

**Files (new):**
- `scenarios/prompt-manager/store/skills/packs/core/report-bug/skill.json`
- `scenarios/prompt-manager/store/skills/packs/core/report-bug/SKILL.md` (per §6.16)

**Tests:** skill registry test — skill loads, lists in `prompt-manager skill discover --query="bug"`.

### Phase H — Create bug-investigator agent identity

**Files (new):**
- `scenarios/prompt-manager/store/agents/bug-investigator/agent.json`
- `scenarios/prompt-manager/store/agents/bug-investigator/SOUL.md`
- `scenarios/prompt-manager/store/agents/bug-investigator/AGENTS.md`
- `scenarios/prompt-manager/store/agents/bug-investigator/TOOLS.md` — bindings explicitly include the investigation-techniques registry skills (currently `scientific-debugging`)

### Phase I — Bind bug-investigator to scenario-qa team

**Files (new):**
- `store/teams/scenario-qa/members/bug-investigator/topics.json` (per §6.12)
- `store/teams/scenario-qa/members/bug-investigator/HEARTBEAT.md`
- `store/teams/scenario-qa/members/bug-investigator/RESPONSIBILITIES.md` — includes "Available Skills" table listing every registered investigation technique
- `store/teams/scenario-qa/members/bug-investigator/last-handoff.md` (initial empty)

### Phase J — Create qa-contrarian agent identity

**Files (new):**
- `scenarios/prompt-manager/store/agents/qa-contrarian/agent.json`
- `scenarios/prompt-manager/store/agents/qa-contrarian/SOUL.md`
- `scenarios/prompt-manager/store/agents/qa-contrarian/AGENTS.md`
- `scenarios/prompt-manager/store/agents/qa-contrarian/TOOLS.md` — bindings: prompt-manager team decision-list/knowledge-list, swarm-manager CLI for backlog-item review

Templated from `marketing-contrarian` (closest analogue).

### Phase K — Bind qa-contrarian to scenario-qa team

**Files (new):**
- `store/teams/scenario-qa/members/qa-contrarian/topics.json` (per §6.13)
- `store/teams/scenario-qa/members/qa-contrarian/HEARTBEAT.md`
- `store/teams/scenario-qa/members/qa-contrarian/RESPONSIBILITIES.md`
- `store/teams/scenario-qa/members/qa-contrarian/last-handoff.md` (initial empty)

### Phase L — Update existing members

**Files (modified):**
- `store/teams/scenario-qa/members/quality-auditor/RESPONSIBILITIES.md` (per §6.14)
- `store/teams/scenario-qa/members/quality-auditor/HEARTBEAT.md` (per §6.14)
- `store/teams/scenario-qa/members/programmatic-qa-runner/RESPONSIBILITIES.md` (per §6.15)
- `store/teams/scenario-qa/members/programmatic-qa-runner/HEARTBEAT.md` (per §6.15)

`topics.json` files reviewed; no structural changes needed (per §6.14, §6.15).

### Phase M — Update scenario-qa team config

**Files (modified):**
- `scenarios/prompt-manager/store/teams/scenario-qa/team.json`:
  - Mission rewritten (per §6.3)
  - Member roster: add `bug-investigator`, `qa-contrarian` to `members{}` block with full `lane`, `ownedDecisionContexts`, `pendingOwnedDecisionCap`, `requiredKnowledgeTopics`, `allowedWrites`, `forbiddenWrites`, `safetyCriticalRules`, `readOnlyModeBehavior`, `newDecisionCapPerHeartbeat`, `taskParameters` blocks
  - `decisionContexts{}` block: add `bug-resolution-proposal` (owner: `bug-investigator`)
  - `knowledgeTopics{}` block: add `bug-investigation-report/<slug>` (owner: `bug-investigator`), `challenge-report/<slug>` (owner: `qa-contrarian`)
  - Bump `revision` (currently 2 → 3) and `updatedAt`
- `scenarios/prompt-manager/store/teams/scenario-qa/roles.json`:
  - Add `bug-investigator` role with description
  - Add `qa-contrarian` role with description
- `scenarios/prompt-manager/store/teams/scenario-qa/shared/TEAM.md`:
  - Mission line updated to match `team.json`
  - Member list updated (add bug-investigator, qa-contrarian with one-line summary)
  - Add `bug-inbox/*` and `bug-investigation-report/*` and `challenge-report/*` to knowledge-topic table
  - Document universal-source pattern + investigation/audit/readiness technique registry pointers
  - Team-specific principles list expanded to mention investigation rigor and contrarian discipline

### Phase N — Update heartbeat Storage Map

**Files (modified):**
- `scenarios/prompt-manager/api/heartbeat/prompt_builder.go`:
  - In Storage Map → Observe, after the existing typed-inbox-vs-notebook rule, add:
    > "For broken scenario or code behavior — bugs — use `prompt-manager skill read report-bug` and follow it; the skill writes to `bug-inbox/*` on `scenario-qa`."

**Tests:** `prompt_builder_test.go` asserts the line appears; heartbeat snapshot for bug-investigator covers Inbox Flow rendering.

### Phase O — Update agent-system PoR

**Files (modified):**
- `docs/agent-system/TOPICS.md`:
  - Update scenario-qa team section in § Per-team topic registry:
    - Add `bug-investigator` row with `bug-inbox/*` intake (taxonomy: bug-report, source: `*`) and `bug-investigation-report/*` output
    - Add `qa-contrarian` row with `challenge-report/*` output
    - Update programmatic-qa-runner and quality-auditor cross-references to new docs
    - Remove "no contrarian" gap from Observations
    - Remove "no inbox" gap from Observations (bug-investigator covers it)
    - Add note about future `qa-inbox/*`/`audit-inbox/*` as workshop-pending if producer emerges
  - Cross-reference to `docs/scenario-qa/investigation-techniques/`, `audit-techniques/`, `readiness-checks/`
- `docs/agent-system/TOPICS_SCHEMA.md`:
  - Document `source_team: "*"` semantics in § Intake entry table
  - New § Universal-source intakes subsection — pattern, validator behavior, when to use
- `docs/agent-system/INTAKE_PIPELINE.md` — note universal-source intakes in § Two routing modes context (they pair with deterministic-prefix routing because the producer set is open and trust-by-construction is needed at the producer side; the investigator's first triage step is to validate the assignment)
- `docs/agent-system/README.md`:
  - Update mental-model paragraph to mention scenario-qa as the bug-triage hub and home of the four (now mature) team PoRs alongside marketing/monetization/agent-system
  - § Active taxonomies registry already updated in Phase F

### Phase P — Enable scenario-qa team

**Files (modified):**
- `scenarios/prompt-manager/store/teams/scenario-qa/team.json` — `enabled: true`

The substrate has been built; now triage and contrarian challenge run at the team's queue cadence. No risk to existing flows because (a) the team has no other active drainers competing for resources at scale, (b) `decisionMode: "yolo"` requires no operator approval per decision, (c) the existing two members (`quality-auditor`, `programmatic-qa-runner`) are proactive and unaffected by the inbox addition.

This step is a deployment decision but the plan recommends including it. Without enabling, bug-inbox accumulates and `piling_inbox` warns within a week — the substrate is half-built.

### Phase Q — Tests + verification

Comprehensive automated coverage; no manual checklists.

| Layer | Test file | Asserts |
|---|---|---|
| Schema | `memberflow/schema_test.go` | `source_team: "*"` parses correctly |
| Validation | `memberflow/validation_test.go` | `orphan_input` skip + `wildcard_source_misuse` fires |
| Taxonomy loader | `memberflow/taxonomy_test.go` | `bug-report` taxonomy loads cleanly |
| Taxonomy authoring | `memberflow/taxonomy_authoring_test.go` | bug-investigator `topics.json` references resolve; both schemas resolve |
| Heartbeat render | `heartbeat/inbox_flow_test.go` | bug-investigator section renders correctly (golden snapshot) |
| Prompt ordering | `heartbeat/prompt_builder_test.go` | Storage Map includes bug-report pointer |
| Skill registry | `cli/skills_test.go` | `report-bug` discoverable; required-reading cites `BUG_REPORT_TAXONOMY.md` |
| Investigation-techniques pairing | extended `agent_system_canon_test.sh` | every `docs/scenario-qa/investigation-techniques/<slug>.md` (excluding README) has matching skill `core/<slug>/`, AND every skill tagged `investigation-technique` has matching PoR doc |
| Audit-techniques pairing | extended `agent_system_canon_test.sh` | every `docs/scenario-qa/audit-techniques/<slug>.md` (excluding README) has matching skill `core/<slug>/`, AND every skill tagged `audit-technique` has matching PoR doc |
| End-to-end | new `bug_inbox_e2e_test.go` | full write → drain cycle works |

Snapshot: `testdata/inbox_flow/scenario-qa__bug-investigator.golden`, `testdata/inbox_flow/scenario-qa__qa-contrarian.golden`. Update tooling: `go test -update`.

## 8. Validation Rules

### Added (memberflow/validation.go)

| Rule | Severity | Condition |
|---|---|---|
| `wildcard_source_misuse` | warning | `intake[].source_team == "*"` AND `external_producers` does not name a documented producer-side anchor (skill or external system) |

### Modified

| Rule | Change |
|---|---|
| `orphan_input` | When `source_team == "*"`, skip producer-existence check |

### Added (canon coherence test)

| Check | Asserts |
|---|---|
| Investigation-techniques pairing | every `docs/scenario-qa/investigation-techniques/<slug>.md` (excluding `README.md`) has a matching `scenarios/prompt-manager/store/skills/packs/core/<slug>/SKILL.md`, AND every skill tagged `tags: ["investigation-technique"]` has a matching PoR doc |
| Audit-techniques pairing | same shape, scoped to `docs/scenario-qa/audit-techniques/` and skills tagged `audit-technique` |

The canon coherence test loops over a list of `{registryDir, skillTag}` pairs so future registries (readiness-checks, future technique types) plug in by appending one entry. Implementation lives in `scenarios/prompt-manager/test/agent_system_canon_test.sh`.

## 9. Tests

All automated. See Phase Q table.

## 10. Migration Order

Strict order — each phase deployable on its own:

1. **A** — Schema + validation (server-only; no agent-visible change)
2. **B** — scenario-qa team PoR README (new docs; no agent-visible change yet)
3. **C** — Investigation-techniques registry + scientific-debugging entry (new docs + skill cross-link)
4. **D** — Audit-techniques registry + 7 paired PoR docs + 7 skill cross-link updates (heaviest authoring phase)
5. **E** — Readiness-checks stub README (lightweight)
6. **F** — `bug-report` taxonomy + `Active taxonomies` registry update
7. **G** — `report-bug` skill
8. **H** — bug-investigator agent identity
9. **I** — bug-investigator team binding
10. **J** — qa-contrarian agent identity
11. **K** — qa-contrarian team binding
12. **L** — Update existing members (RESPONSIBILITIES + HEARTBEAT cross-links)
13. **M** — scenario-qa `team.json` + `roles.json` + `shared/TEAM.md` update (mission rewrite, member roster, decision contexts, knowledge topics)
14. **N** — Storage Map prompt update (every agent's next heartbeat now sees the bug-report pointer)
15. **O** — agent-system PoR updates (TOPICS.md, TOPICS_SCHEMA.md, INTAKE_PIPELINE.md, README.md)
16. **P** — Enable scenario-qa team
17. **Q** — Tests + verification

A producer can write a bug-inbox entry as soon as Phases A, F, G land. The investigator drains it as soon as Phases A, F, G, H, I, M land. The contrarian starts producing as soon as J, K, M land. Substrate gracefully degrades: if execution stalls between O and P, entries accumulate and `piling_inbox` warns the operator.

Phases C and D can technically execute in parallel (no shared files) but the plan recommends serial execution to keep PRs small and reviewable.

## 11. Cleanup & Health Verification

Per project feedback memory:

1. **Fix all lint, type, and unit-test issues in modified files — including pre-existing.**
   - Go: `cd scenarios/prompt-manager/api && gofumpt -w . && golangci-lint run && go test ./... -timeout 300s`
2. **Restart the scenario.** `vrooli scenario restart prompt-manager`
3. **Verify health.**
   - `prompt-manager graph topics` returns 0 errors; shows `bug-inbox/*` with `source_team: "*"`, `bug-investigation-report/*`, `challenge-report/*` for scenario-qa
   - `prompt-manager graph drain-status` shows scenario-qa/bug-investigator
   - `prompt-manager team member-context scenario-qa bug-investigator` renders Inbox Flow correctly
   - `prompt-manager team member-context scenario-qa qa-contrarian` renders proactive section correctly
   - `prompt-manager team member-context scenario-qa quality-auditor` includes Available Skills table referencing audit-techniques
   - `prompt-manager skill discover --query="bug"` returns `report-bug`
   - `bash scenarios/prompt-manager/test/agent_system_canon_test.sh` passes (investigation + audit pairing both green)

## 12. Risks

1. **scenario-qa README is full but technique registries are uneven.** Investigation has one entry, audit has seven, readiness has zero. Mitigation: each registry's README declares its current state and graduation rules; the asymmetry is intentional (audit has the most mature skill base; readiness is least mature) and documented.
2. **Audit-technique PoR-doc authoring is the heaviest single phase.** Seven docs at once is real authoring work. Mitigation: each doc follows the same template (definition, when-applies, when-backfires, contrarian-failure-modes, skill citation); content draws from the existing skills' SKILL.md files; doc length expected to be ~50-100 lines each, not exhaustive treatises.
3. **Universal-source intake is a new pattern; misuse risk.** Easy to slap `source_team: "*"` on something that should have specific producers. Mitigation: `wildcard_source_misuse` warning + the new TOPICS_SCHEMA.md subsection.
4. **bug-investigator overwhelmed by volume.** Single member draining cross-team intake. Mitigation: queue policy on `team.json` caps maxConcurrentRuns; severity=blocker entries jump the queue via heartbeat task logic. If volume requires it later, splitting into specialized investigators (e.g., infra-bug-investigator, scenario-bug-investigator) is a follow-up plan.
5. **qa-contrarian over-challenges or under-challenges.** New contrarian; calibration takes time. Mitigation: HEARTBEAT.md caps challenge-notes per heartbeat; SOUL.md emphasizes "honest when no challenge is warranted"; first month of challenge-notes reviewed by operator.
6. **Misclassification at write time.** Producer picks wrong signal-type. Mitigation: deterministic-prefix routing means investigator validates as first triage step (per §6.11); `route-to-another-topic` handles legitimate misclassification. Recurring misclassification on a specific producer is a `meta-self-improvement` decision (improve `report-bug` skill clarity).
7. **bug-investigation log topic isn't drained.** It's an audit log, not an inbox. Mitigation: documented as intentional in TOPICS.md § Topic shapes; `orphan_output` warning is by-design here, same as `quality-audit/*` and other audit logs.
8. **`report-bug` skill on-demand discoverability.** If agents don't know to look for it, bugs still go to notebook. Mitigation: Storage Map points to it explicitly in every agent's heartbeat; over time, observed mismatch (bugs in notebook that should have been in bug-inbox) gets flagged by the curator and routed correctly.
9. **Technique-registry growth.** Adding new techniques requires `meta-self-improvement` decisions, paired doc + skill, registry table updates. Mitigation: same lifecycle the marketing post-techniques registry uses; the canon coherence test enforces pairing.
10. **`challenge-report/*` orphan-output applies to qa-contrarian like every other contrarian.** No drainer for cross-team contrarian writes. Mitigation: documented in TOPICS.md known inconsistency #3; out of scope for this plan; revisited as a future workshop decision (e.g., `meta-contrarian` cross-drains peer `challenge-report/*`).

## 13. Open Questions

All architectural decisions in this plan are settled. The remaining items are genuinely future scope, not deferrals:

1. **Cross-cutting drain of `challenge-report/*`.** From TOPICS.md known inconsistency #3. Plausible that one team's contrarian (e.g., meta-contrarian) drains every team's `challenge-report/*` cross-team, but out of scope; revisit as a future workshop decision.
2. **`qa-inbox/*` / `audit-inbox/*` for existing members.** Operator-fed alpha for QA review and deep audits. Currently no producer; would orphan_input. Document as future PoR work; revisit when (e.g.) `vision-walk-prep` adds them as output prefixes.
3. **Filling out scenario-qa's full PoR** beyond the README + three registries. Quality principles, scenario-classification heuristics — flagged as future operator-curated decisions in the team README's "Future PoR work" section.
4. **Future investigation techniques.** `scientific-debugging` is the only registered technique at landing time. Future candidates the bug-investigator's audit log will surface graduation candidates for: bisect-debugging (binary-search git history), minimal-reproduction (reduce complex case to smallest repro), differential-trace (compare working vs broken), comparative-environments (test in different envs), 5-whys, fishbone analysis. Each enters via `meta-self-improvement` decision.
5. **Future audit techniques.** Beyond the seven existing skills: performance-audit, security-audit, deprecation-audit, accessibility-audit, observability-audit. Same graduation flow.
6. **Readiness-checks population.** Once GCT dimensions stabilize (or are replaced by an internal Vrooli equivalent), individual readiness checks graduate to paired doc + skill in `readiness-checks/`.

---

This plan is greenfield, single-implementation (no v1/v2 splits), and replicates the doc-and-paired-skill discipline marketing established as the team-PoR gold standard — extended across three parallel registries (investigation, audit, readiness). Execution follows Phases A→Q in order; verification gates between each phase.
