# Bug-Inbox + Scenario-QA Plan-of-Record: Plan of Action

Status: draft. Authored 2026-05-03. Will move out of `drafts/` once execution begins.

This is the proper long-term implementation: scenario-qa team gets its own plan-of-record, an investigation-techniques registry (paired-doc-and-skill, mirroring marketing's `post-techniques/` pattern), a `bug-investigator` member that drains a cross-team `bug-inbox/*`, and the schema upgrade required to make universal-source intakes a first-class concept.

## 1. Required Reading

```bash
prompt-manager skill read decision-boundary-extraction seam-discovery-and-enforcement boundary-of-responsibility-enforcement scientific-debugging team-shared-docs-pattern
```

Plus PoR files governing the substrate being changed:

- `docs/agent-system/INTAKE_PIPELINE.md` (two routing modes; cross-team schema ownership)
- `docs/agent-system/TOPICS.md` (registry; notebook-vs-typed-inbox)
- `docs/agent-system/TOPICS_SCHEMA.md` (schema; cross-team flow)
- `docs/agent-system/PRIMITIVES.md`
- `docs/agent-system/LAYERS.md`
- `docs/agent-system/TEAM_DOCS_PATTERNS.md` — paired-doc-and-skill discipline
- `docs/agent-system/PROMOTION_LADDER.md` — how techniques mature
- `docs/marketing/post-techniques/README.md` — the gold-standard reference for the technique-registry pattern this plan replicates
- `docs/marketing/post-types/README.md` — the doc + paired skill rule (mandatory hard rule, not recommendation)
- `scenarios/prompt-manager/api/memberflow/schema.go` and `validation.go`
- `scenarios/prompt-manager/api/heartbeat/inbox_flow.go` and `prompt_builder.go`
- `scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md`
- `scenarios/prompt-manager/store/agents/quality-auditor/` (agent template reference)
- `scenarios/prompt-manager/store/teams/scenario-qa/team.json` (note `enabled: false`)

## 2. Context

Decision #2 in the post-inbox-flow workshop concluded: **add a cross-team `bug-inbox/*` on `scenario-qa`, drained by a new dedicated `bug-investigator` member.** The producer/consumer split exists because bugs are observation-shaped, not proposal-shaped — the agent that discovers a bug usually doesn't have the context (or the time mid-task) to investigate it.

Three parallel substrate gaps surfaced during planning:

1. **scenario-qa has no plan-of-record.** Other domain teams (marketing, monetization, agent-system) own a `docs/<domain>/` folder of operator-curated canon. scenario-qa does not, which means every bug investigation reinvents the methodology. Adding a `bug-investigator` member without a team PoR would compound the gap, not close it.
2. **No registry of investigation techniques.** `scientific-debugging` is the gold-standard skill for root-cause analysis but lives as a stand-alone skill with no paired strategic-canon doc. Other techniques the team will eventually need (bisect-debugging, minimal-repro, differential-trace, comparative-environments) have nowhere to land. The marketing-team `post-techniques/` pattern is the right reference: one PoR doc per technique, paired with one skill, organized in a registry folder with documented graduation rules.
3. **No universal-source intake pattern.** `bug-inbox/*` will receive writes from every team's members — a topology not currently supported by the validator. A real schema upgrade is required (`source_team: "*"`).

This plan addresses all three together. They share the same execution boundary (one team's substrate), and shipping the bug-inbox without the team PoR would leave the bug-investigator with no canonical home for its growing technique library.

## 3. Goals

1. **Universal bug capture.** Any agent on any team records a bug observation in one skill invocation, regardless of which scenario, skill, or component is broken.
2. **Specialist triage with extensible methodology.** A new `bug-investigator` member on scenario-qa drains the inbox using the technique registry. The first technique is `scientific-debugging`; new techniques graduate into the registry over time via standard meta-optimization decisions.
3. **scenario-qa plan-of-record.** A new `docs/scenario-qa/` folder containing the team's strategic canon: README, investigation-techniques registry, bug-report taxonomy. Operator-curated like every other team PoR.
4. **Investigation-techniques registry.** `docs/scenario-qa/investigation-techniques/` mirrors the marketing `post-techniques/` pattern — one paired doc-and-skill per technique, registry README documenting graduation rules. `scientific-debugging.md` is the first entry, paired with the existing skill.
5. **Schema upgrade for universal-source intakes.** `intake[].source_team: "*"` becomes a first-class declaration meaning "any team's members may write." Validator handles correctly. Documented in TOPICS_SCHEMA.md.
6. **bug-report taxonomy.** Six signal types, schema for entries, action-selection rules, evidence rules. JSON sidecar + markdown PoR pair.
7. **Storage Map integration.** Every agent's heartbeat points to `report-bug` as the canonical write-side skill for bug observations.
8. **Live triage.** scenario-qa team enabled; bug-investigator runs at the team's queue cadence.
9. **Zero net behavior change for other members.** Existing inboxes, taxonomies, classifiers, members, and decisions are not affected.

## 4. Non-Goals

- **Other observation inboxes (capability-gap, friction).** Already handled by existing decision contexts and the Storage Map fan-out. Out of scope.
- **Authoring additional investigation techniques.** Only `scientific-debugging` graduates into the registry as part of this plan; future techniques follow the standard `meta-self-improvement` decision flow.
- **Migrating existing notebook entries that are bugs.** Bugs already in notebooks stay there; the curator promotes/retires per usual notebook-debt rules. New bugs go to bug-inbox.
- **Fleshing out the rest of scenario-qa's PoR.** This plan creates the README skeleton and the parts needed for bug investigation. Other strategic canon (e.g., quality principles, audit dimensions used by quality-auditor) is documented as future PoR work, not authored here.
- **Splitting bug-investigator into specialist sub-investigators.** Single member; if volume requires it later, splitting is a follow-up plan.

## 5. Greenfield Statement

**This is greenfield work, end-to-end.** New team PoR folder, new technique registry, new taxonomy, new agent identity, new member binding, new writer skill, new schema field, new validator behavior. No backwards-compatibility shims, no aliases, no v1/v2 deferrals — every architectural decision is settled in §6 and implemented in §7. After this lands, the scenario-qa team has a complete operating surface (team PoR, member roster, technique registry, bug-inbox drain duty) by the same standard the marketing and monetization teams meet today.

## 6. Architecture Summary

### 6.1 Layer table

| Concern | Home |
|---|---|
| Team mission, scope, principles | `docs/scenario-qa/README.md` (new — strategic canon) |
| Bug-report capture procedure | `report-bug` skill (writer skill, opt-in via Storage Map pointer) |
| Bug-report classification | Deterministic-prefix routing — producer picks signal-type; investigator validates as the first step of investigation. No separate classifier skill. (See §6.6 for rationale.) |
| Investigation methodology canon | `docs/scenario-qa/investigation-techniques/<slug>.md` (one PoR doc per technique) |
| Investigation methodology procedure | `scenarios/prompt-manager/store/skills/packs/core/<technique-slug>/SKILL.md` (one skill per technique; doc + paired skill discipline) |
| Drain procedure | `scenario-qa/bug-investigator` member's heartbeat — Inbox Flow section generated from topics.json |
| Triage outcomes | bug-report taxonomy's `actionSelection` (drop / observe / file-backlog / file-decision / route-to-another-topic / capability-gap) |
| Universal-source declaration | `intake[].source_team: "*"` in topics.json (new schema semantics) |
| Bug investigation audit log | `bug-investigation/<slug>` knowledge entries (one per closed bug — root cause, technique applied, action taken) |

### 6.2 Topic shape

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

### 6.3 Front-matter schema

Every `bug-inbox/*` entry begins with:

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

### 6.4 Schema change: `source_team: "*"`

`IntakeEntry.source_team` accepts:

- `null` — same-team source or external producer (existing)
- `<team-id>` — specific cross-team source (existing)
- `"*"` (new) — universal: any team's members may write. Validator skips `orphan_input` for the entry.

Documented in `TOPICS_SCHEMA.md` § Cross-team flow + new § Universal-source intakes subsection.

### 6.5 scenario-qa plan-of-record

New folder `docs/scenario-qa/`. Skeleton authored in this plan; expansion via future operator-curated decisions.

```
docs/scenario-qa/
├── README.md                              # team PoR overview
├── investigation-techniques/
│   ├── README.md                          # registry; lifecycle rules
│   └── scientific-debugging.md            # paired with existing skill
├── bug-report-taxonomy.json               # JSON sidecar
└── BUG_REPORT_TAXONOMY.md                 # human-readable view
```

`docs/scenario-qa/README.md` documents:

- Mission (cross-link to `store/teams/scenario-qa/shared/TEAM.md`)
- Scope and boundaries
- Folder map of the PoR (techniques registry, taxonomies)
- Decision contexts owned by the team
- Cross-team relationships (e.g., bug-resolution-proposal → swarm-manager backlog)
- Future PoR work explicitly noted (quality principles, audit dimensions, etc.) — flagged as gaps to fill in later operator-curated decisions, not authored here

### 6.6 Investigation-techniques registry

Mirrors `docs/marketing/post-techniques/` exactly:

- **Doc + paired skill discipline (mandatory).** Every technique ships as a strategic-canon doc *and* an executable skill. A doc with no skill is a stale shrine; a skill with no doc is brittle. Both halves required, no exceptions.
- **One technique, one folder location.** Cross-cutting techniques get a single canonical home; bug-investigator references each technique it applies.
- **Lifecycle: v0 (doc-only stub) → v1 (doc + skill, active).** v0 means strategic canon is documented but the technique is not yet active in production. Activation requires (1) skill is authored, (2) skill cites the technique doc as required reading, (3) doc Status line bumped to v1, (4) `bug-investigator/RESPONSIBILITIES.md` references the skill in its Available Skills table.
- **Compression operates per-skill.** Each technique's skill compresses independently as Vrooli's substrate (CLIs, debug tooling) absorbs more of the work. A unified `bug-investigate` mega-skill that branches on technique would compress worse — same argument the marketing team uses for one skill per post type.
- **Adding a technique.** New techniques enter via `meta-self-improvement` decision proposing the addition. The bug-investigator surfaces graduation candidates from observed patterns in `bug-investigation/<slug>` audit entries.

`scientific-debugging.md` is the registry's first entry. The existing `scientific-debugging` skill is updated to add `docs/scenario-qa/investigation-techniques/scientific-debugging.md` to its required-reading list.

### 6.7 No classifier skill — deterministic-prefix routing (settled)

Bug-inbox uses deterministic-prefix routing without a classifier skill. The producer assigns the signal-type at write time (via the `report-bug` skill, which prompts for it from a fixed list); the bug-investigator validates the assignment as the first step of investigation.

This matches the notebook-debt pattern (curator does classification as part of curation), not the marketing-research pattern (separate classifier skill). The reason: **bug investigation is fundamentally one workflow that includes classification as a sub-step**. Separating classification into its own skill creates a hop without value because the investigator must read the entry anyway to start investigation. The classifier's hypothetical role (assign signal-type, evidence-strength, honesty-flags) is identical to the first three steps of investigation.

This is not a deferral. Adding a classifier later would be architectural drift, not an upgrade.

### 6.8 `bug-investigator` member

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
    { "prefix": "bug-investigation/*", "destination_kind": "knowledge", "destination_team": null }
  ],
  "decisions_owned": ["bug-resolution-proposal"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["any-team-member-via-report-bug-skill"]
}
```

`bug-investigation/<slug>` is the durable audit log — one entry per closed bug, capturing the technique applied, root cause, and action taken. Used to surface graduation candidates for new investigation techniques and to detect repeat bugs.

`bug-resolution-proposal` decision context covers cross-cutting fixes that require operator approval (e.g., "rename this CLI verb because three bugs trace to its ambiguous name").

### 6.9 Member files

Created from the quality-auditor template:

- `store/agents/bug-investigator/SOUL.md` — identity (specialist scientific-debugger; calm, methodical, no speculation; honest when investigation is blocked)
- `store/agents/bug-investigator/AGENTS.md` — workflow contract (load technique skill from registry, drain top of inbox, investigate, classify outcome, write `bug-investigation/<slug>` log, hand off via swarm-manager / decision / route)
- `store/agents/bug-investigator/TOOLS.md` — bindings: every technique skill from the registry (currently `scientific-debugging`), swarm-manager CLI, prompt-manager team knowledge-list/update/delete (no add — bug-investigator never writes to its own inbox)
- `store/agents/bug-investigator/agent.json` — metadata
- `store/teams/scenario-qa/members/bug-investigator/topics.json` — above
- `store/teams/scenario-qa/members/bug-investigator/HEARTBEAT.md` — heartbeat task: drain top of inbox by severity, pick technique from registry, investigate, write audit-log entry, take action
- `store/teams/scenario-qa/members/bug-investigator/RESPONSIBILITIES.md` — drain duty + investigation rigor + escalation rules + Available Skills table listing every technique
- `store/teams/scenario-qa/members/bug-investigator/last-handoff.md` — initialized empty

### 6.10 `report-bug` skill

Universal writer skill any agent can invoke. Lives in the `core` skill pack; discovered via `prompt-manager skill discover --query="bug"` and pointed-to from the heartbeat Storage Map.

- `store/skills/packs/core/report-bug/skill.json` — id `report-bug`, modes `["tools"]`, tags `["observability", "bug-report"]`
- `store/skills/packs/core/report-bug/SKILL.md`:
  - Required reading: `docs/scenario-qa/BUG_REPORT_TAXONOMY.md`
  - Inputs: signal-type (from fixed list), severity, repro, expected, actual, context, description
  - Procedure: validate inputs against schema; generate kebab-case slug; construct `bug-inbox/<signal-type>/<slug>`; format front-matter + body; invoke `prompt-manager team knowledge-add scenario-qa --by=<reporter-agent-id> --topic="..." --content="..."`
  - Output: confirms entry id

The skill is destination-coupled by design (writer skills always are; the `non_portable_classifier` rule applies only to classifier skills).

**Auto-loading vs. on-demand.** The skill is **on-demand** — discovered via the Storage Map pointer, loaded when an agent decides to report a bug. Auto-loading into every heartbeat would bloat agent context with rarely-used procedure. The Storage Map pointer is the discoverability anchor; the skill carries the procedure.

### 6.11 `bug-report` taxonomy

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
    "observe": "Confirmed bug; route findings to `bug-investigation/<slug>` and continue (no fix required this heartbeat).",
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

**Goal:** topics.json supports `source_team: "*"`; validator handles correctly. No agent-visible change yet.

**Files:**
- `scenarios/prompt-manager/api/memberflow/schema.go` — `IntakeEntry.SourceTeam` semantics extended; `"*"` literal accepted
- `scenarios/prompt-manager/api/memberflow/validation.go`:
  - `orphan_input` rule: when `source_team == "*"`, skip producer-existence check
  - New rule `wildcard_source_misuse` (warning): `source_team == "*"` AND `external_producers` does not name any anchor — catches "I made it universal but forgot to document who actually writes"

**Tests:** schema_test.go (parsing), validation_test.go (positive and negative cases for both rule changes).

### Phase B — Author scenario-qa team plan-of-record

**Files (new):**
- `docs/scenario-qa/README.md` — team PoR overview
  - Mission, scope, boundaries (cross-link to `store/teams/scenario-qa/shared/TEAM.md`)
  - Folder map (techniques registry, taxonomies, future PoR sections)
  - Decision contexts owned (`bug-resolution-proposal`, `deep-audit-backlog`, `preemptive-qa-backlog`)
  - Cross-team relationships
  - "Future PoR work" section explicitly listing what's not yet authored: quality principles, audit dimensions, scenario classification heuristics, etc. — flagged as gaps for future operator-curated decisions

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
  - Strategic canon: when this technique applies, when it backfires, what failure modes the investigator watches for
  - Cites the existing skill as the executable spec
  - Cites `docs/agent-system/SKILL_AUTHORING.md`

**Files (modified):**
- `scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md` — add `docs/scenario-qa/investigation-techniques/scientific-debugging.md` to required reading; cross-link the strategic canon

### Phase D — Author bug-report taxonomy

**Files (new):**
- `docs/scenario-qa/bug-report-taxonomy.json` (per §6.11)
- `docs/scenario-qa/BUG_REPORT_TAXONOMY.md` — human-readable view; cite JSON sidecar; explain signal types with examples

**Tests:** `taxonomy_authoring_test.go` extension — taxonomy loads; both schema ids (`bug-report`, `bug-investigation`) resolve; `defaultMethod: "scientific-debugging"` resolves to a registered skill.

### Phase E — Author `report-bug` skill

**Files (new):**
- `scenarios/prompt-manager/store/skills/packs/core/report-bug/skill.json`
- `scenarios/prompt-manager/store/skills/packs/core/report-bug/SKILL.md` (per §6.10)

**Tests:** skill registry test — skill loads, lists in `prompt-manager skill discover --query="bug"`.

### Phase F — Create bug-investigator agent identity

**Files (new):**
- `scenarios/prompt-manager/store/agents/bug-investigator/agent.json`
- `scenarios/prompt-manager/store/agents/bug-investigator/SOUL.md`
- `scenarios/prompt-manager/store/agents/bug-investigator/AGENTS.md`
- `scenarios/prompt-manager/store/agents/bug-investigator/TOOLS.md` — bindings explicitly include the investigation-techniques registry skills (currently `scientific-debugging`)

### Phase G — Bind bug-investigator to scenario-qa team

**Files (new):**
- `store/teams/scenario-qa/members/bug-investigator/topics.json` (per §6.8)
- `store/teams/scenario-qa/members/bug-investigator/HEARTBEAT.md`
- `store/teams/scenario-qa/members/bug-investigator/RESPONSIBILITIES.md` — includes "Available Skills" table listing every registered investigation technique
- `store/teams/scenario-qa/members/bug-investigator/last-handoff.md` (initial empty)

### Phase H — Update scenario-qa team config

**Files (modified):**
- `scenarios/prompt-manager/store/teams/scenario-qa/team.json`:
  - Add `bug-investigator` to member roster
  - Add `bug-resolution-proposal` to ownedDecisionContexts
  - Bump revision and updatedAt
- `scenarios/prompt-manager/store/teams/scenario-qa/roles.json` — declare `bug-investigator` role
- `scenarios/prompt-manager/store/teams/scenario-qa/shared/TEAM.md`:
  - Add bug-investigator to member list
  - Add `bug-inbox/*` to knowledge-topic-taxonomy table
  - Document universal-source pattern + investigation-techniques registry pointer

### Phase I — Update heartbeat Storage Map

**Files (modified):**
- `scenarios/prompt-manager/api/heartbeat/prompt_builder.go`:
  - In Storage Map → Observe, after the existing typed-inbox-vs-notebook rule, add:
    > "For broken scenario or code behavior — bugs — use `prompt-manager skill read report-bug` and follow it; the skill writes to `bug-inbox/*` on `scenario-qa`."

**Tests:** `prompt_builder_test.go` asserts the line appears; heartbeat snapshot for bug-investigator covers Inbox Flow rendering.

### Phase J — Update agent-system PoR

**Files (modified):**
- `docs/agent-system/TOPICS.md`:
  - Add `bug-inbox/*` to scenario-qa registry
  - Add `bug-investigation/*` to scenario-qa registry (Local audit log shape)
  - Update scenario-qa observations: remove "no inbox" gap; add bug-investigator
  - Cross-reference to `docs/scenario-qa/investigation-techniques/`
- `docs/agent-system/TOPICS_SCHEMA.md`:
  - Document `source_team: "*"` semantics in § Intake entry table
  - New § Universal-source intakes subsection — pattern, validator behavior, when to use
- `docs/agent-system/INTAKE_PIPELINE.md` — note universal-source intakes in § Two routing modes context (they pair with deterministic-prefix routing because the producer set is open and trust-by-construction is needed at the producer side; the investigator's first triage step is to validate the assignment)
- `docs/agent-system/README.md`:
  - Add `bug-report` to § Active taxonomies registry
  - Update mental-model paragraph to mention scenario-qa as the bug-triage hub

### Phase K — Enable scenario-qa team

**Files (modified):**
- `scenarios/prompt-manager/store/teams/scenario-qa/team.json` — `enabled: true`

The substrate has been built; now triage runs at the team's queue cadence. No risk to existing flows because (a) the team has no other active drainers competing for resources, (b) `decisionMode: "yolo"` requires no operator approval per decision, (c) the only existing members (`quality-auditor`, `programmatic-qa-runner`) are proactive and unaffected.

This step is a deployment decision but the plan recommends including it. Without enabling, bug-inbox accumulates and `piling_inbox` warns within a week — the substrate is half-built.

### Phase L — Tests + verification

Comprehensive automated coverage; no manual checklists.

| Layer | Test file | Asserts |
|---|---|---|
| Schema | `memberflow/schema_test.go` | `source_team: "*"` parses correctly |
| Validation | `memberflow/validation_test.go` | orphan_input skip + wildcard_source_misuse fires |
| Taxonomy loader | `memberflow/taxonomy_test.go` | bug-report taxonomy loads cleanly |
| Taxonomy authoring | `memberflow/taxonomy_authoring_test.go` | bug-investigator topics.json references resolve; both schemas resolve |
| Heartbeat render | `heartbeat/inbox_flow_test.go` | bug-investigator section renders correctly (golden snapshot) |
| Prompt ordering | `heartbeat/prompt_builder_test.go` | Storage Map includes bug-report pointer |
| Skill registry | `cli/skills_test.go` | `report-bug` discoverable; required-reading cites BUG_REPORT_TAXONOMY.md |
| Investigation techniques registry | new `docs_consistency_test.sh` (or extend existing canon test) | every technique under `docs/scenario-qa/investigation-techniques/<slug>.md` has a paired skill at `store/skills/packs/core/<slug>/SKILL.md`; vice versa for skills marked as investigation techniques |
| End-to-end | new `bug_inbox_e2e_test.go` | full write → drain cycle works |

Snapshot: `testdata/inbox_flow/scenario-qa__bug-investigator.golden`. Update tooling: `go test -update`.

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
| Investigation-techniques pairing | every `docs/scenario-qa/investigation-techniques/<slug>.md` (excluding README.md) has a matching `scenarios/prompt-manager/store/skills/packs/core/<slug>/SKILL.md`, AND every skill flagged with `tags: ["investigation-technique"]` has a matching PoR doc |

## 9. Tests

All automated. See Phase L table.

## 10. Migration Order

Strict order — each phase deployable on its own:

1. **A** — Schema + validation (server-only; no agent-visible change)
2. **B** — scenario-qa team PoR creation (new docs; no agent-visible change yet)
3. **C** — Investigation-techniques registry + scientific-debugging entry (new docs + skill cross-link)
4. **D** — bug-report taxonomy
5. **E** — `report-bug` skill
6. **F** — bug-investigator agent identity
7. **G** — Team binding
8. **H** — scenario-qa team config update
9. **I** — Storage Map prompt update (every agent's next heartbeat now sees the bug-report pointer)
10. **J** — agent-system PoR updates
11. **K** — Enable scenario-qa team
12. **L** — Tests + verification

A producer can write a bug-inbox entry as soon as Phases A-E land. The investigator drains it as soon as Phases A-K land. Substrate gracefully degrades: if execution stalls between J and K, entries accumulate and `piling_inbox` warns the operator.

## 11. Cleanup & Health Verification

Per project feedback memory:

1. **Fix all lint, type, and unit-test issues in modified files — including pre-existing.**
   - Go: `cd scenarios/prompt-manager/api && gofumpt -w . && golangci-lint run && go test ./... -timeout 300s`
2. **Restart the scenario.** `vrooli scenario restart prompt-manager`
3. **Verify health.**
   - `prompt-manager graph topics` returns 0 errors; shows `bug-inbox/*` with `source_team: "*"`
   - `prompt-manager graph drain-status` shows scenario-qa/bug-investigator
   - `prompt-manager team member-context scenario-qa bug-investigator` renders Inbox Flow correctly
   - `prompt-manager skill discover --query="bug"` returns `report-bug`
   - `bash scenarios/prompt-manager/test/agent_system_canon_test.sh` passes
   - Investigation-techniques pairing test passes

## 12. Risks

1. **scenario-qa team PoR is a skeleton.** README is authored but the rest of the team's strategic canon (quality principles, audit dimensions used by quality-auditor, scenario classification heuristics) is explicitly future work. Mitigation: README "Future PoR work" section names the gaps so operator-curated decisions can fill them; `quality-auditor` continues operating on `team.json` audit lenses for now.
2. **Universal-source intake is a new pattern; misuse risk.** Easy to slap `source_team: "*"` on something that should have specific producers. Mitigation: `wildcard_source_misuse` warning + the new TOPICS_SCHEMA.md subsection.
3. **bug-investigator overwhelmed by volume.** Single member draining cross-team intake. Mitigation: queue policy on team.json caps maxConcurrentRuns; severity=blocker entries jump the queue via heartbeat task logic. If volume requires it later, splitting into specialized investigators (e.g., infra-bug-investigator, scenario-bug-investigator) is a follow-up plan.
4. **Misclassification at write time.** Producer picks wrong signal-type. Mitigation: deterministic-prefix routing means investigator validates as first triage step (per §6.7); `route-to-another-topic` handles legitimate misclassification. Recurring misclassification on a specific producer is a `meta-self-improvement` decision (improve `report-bug` skill clarity).
5. **bug-investigation log topic isn't drained.** It's an audit log, not an inbox. Mitigation: documented as intentional in TOPICS.md § Topic shapes; `orphan_output` warning is by-design here, same as `quality-audit/*` and other audit logs.
6. **`report-bug` skill on-demand discoverability.** If agents don't know to look for it, bugs still go to notebook. Mitigation: Storage Map points to it explicitly in every agent's heartbeat; over time, observed mismatch (bugs in notebook that should have been in bug-inbox) gets flagged by the curator and routed correctly.
7. **Investigation-techniques registry growth.** Adding new techniques requires meta-self-improvement decisions, paired doc + skill, registry table updates. Mitigation: same lifecycle the marketing post-techniques registry uses; the canon coherence test enforces pairing.

## 13. Open Questions

All architectural decisions in this plan are settled. The remaining items are genuinely future scope, not deferrals:

1. **Cross-cutting drain of `challenge-note/*`.** From the original known-inconsistency #3 in TOPICS.md. Plausible that bug-investigator could also drain peer teams' `challenge-note/*`, but out of scope for this plan; revisit as a future workshop decision.
2. **Filling out scenario-qa's full PoR** beyond the README skeleton. Quality principles, audit dimensions, scenario classification heuristics — all flagged as future operator-curated decisions in the team README's "Future PoR work" section.
3. **Future investigation techniques.** `scientific-debugging` is the only registered technique at landing time. Future candidates that the bug-investigator's audit log will surface graduation candidates for include: bisect-debugging (binary-search git history), minimal-reproduction (reduce complex case to smallest repro), differential-trace (compare working vs broken), comparative-environments (test in different envs), 5-whys, fishbone analysis. Each enters via `meta-self-improvement` decision; not part of this plan.

---

This plan is greenfield, single-implementation (no v1/v2 splits), and replicates the doc-and-paired-skill discipline marketing established as the team-PoR gold standard. Execution follows Phases A→L in order; verification gates between each phase.
