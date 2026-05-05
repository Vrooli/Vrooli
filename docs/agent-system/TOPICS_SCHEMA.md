# `topics.json` — per-member message-flow declarations

**Status:** canon. This document is the plan-of-record for the `topics.json` data layer — Pillar 1 of [topic validation](PRIMITIVES.md#three-pillars-of-topic-validation). It pairs with the Go implementation at `scenarios/prompt-manager/api/memberflow/schema.go`. Cross-team-readable; cited by `INTAKE_PIPELINE.md`, `LAYERS.md`, and the heartbeat builder. Sibling pillars: [`PROSE_SCAN_TARGETS.md`](PROSE_SCAN_TARGETS.md) (P2 — prose scan) and [`RUNTIME_ATTRIBUTION.md`](RUNTIME_ATTRIBUTION.md) (P3 — observed writes).

Promoted from `drafts/topics-schema.md` after the inbox-flow refactor stabilized the schema across five in-prod adopters. Backwards-incompatible changes from this point require a `meta-optimization` decision and a migration plan (see [Stability gate](#stability-gate)).

## Purpose

A `topics.json` file declares, **structurally**, how a single team member produces and consumes work via topic-prefixed channels. It is the machine-readable substrate the heartbeat builder uses to render the universal `# Inbox Flow` section, and the validator uses to detect orphan flows and drift.

Once this layer ships, prose claims like "the researcher drains `research-inbox/*`" become declarations the system can validate, visualize, and lint against. Orphan output prefixes, dangling intake claims, conflicting drain duty, and stalled inboxes become detectable.

`topics.json` is **per-member**, not per-team or per-skill. The topic declarations live at the same granularity as `RESPONSIBILITIES.md` and `HEARTBEAT.md`.

## File location

```
scenarios/prompt-manager/store/teams/<team>/members/<member>/topics.json
```

Sibling to `HEARTBEAT.md`, `RESPONSIBILITIES.md`, `last-handoff.md`. One file per member.

## Schema (canonical)

```json
{
  "intake": [
    {
      "prefix": "research-inbox/*",
      "taxonomy": "marketing-research",
      "classifier_skill": "marketing-signal-classifier",
      "source_team": null
    }
  ],
  "output": [
    {
      "prefix": "audience-scan/*",
      "destination_kind": "knowledge",
      "destination_team": null,
      "schema": "audience-scan"
    },
    {
      "prefix": "monetization-benchmark-adjacent-record/*",
      "destination_kind": "knowledge",
      "destination_team": "monetization",
      "schema": "monetization-benchmark-adjacent"
    }
  ],
  "decisions_owned": ["audience-update", "channel-strategy-update"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["vision-walk", "operator"]
}
```

Top-level keys, all optional (omit when not applicable):

| Key | Type | Meaning |
|---|---|---|
| `intake` | array of intake entries | Topic-prefixes this member drains. Each entry references a taxonomy and (optionally) a classifier skill. |
| `output` | array of output entries | Topic-prefixes this member writes. Each entry names destination kind, optional cross-team destination, and optional front-matter schema. |
| `decisions_owned` | array of decision-context strings | Decision contexts this member owns. |
| `decisions_consumed` | array of decision-context strings | Decision contexts whose acceptance changes this member's behavior. |
| `raises_capability_gaps` | boolean | True if the member's role includes filing `capability-gap` decisions. |
| `external_producers` | array of stable identifiers | Non-team-member producers that feed intake (e.g., `vision-walk`, `operator`, `bookmark-intelligence-hub`). |

### Intake entry

```jsonc
{
  "prefix": "research-inbox/*",
  "taxonomy": "marketing-research",
  "classifier_skill": "marketing-signal-classifier",
  "source_team": null
}
```

| Field | Type | Required | Meaning |
|---|---|---|---|
| `prefix` | string with optional `*` suffix | yes | Topic-prefix matched against `team knowledge-list --topic-prefix=`. |
| `taxonomy` | string | yes | The id of the taxonomy JSON sidecar (`docs/<domain>/<id>.json`) that owns this prefix's signal vocabulary, dispatch, evidence rules, and destination schemas. The validator's `unknown_taxonomy` rule resolves this against the registry; an empty value fires `missing_taxonomy`. |
| `classifier_skill` | string | optional | A portable, member-agnostic judgment skill that assigns `signal_type` from the taxonomy. Optional: when the topic-prefix carries a deterministic signal type, no classifier is needed. The `non_portable_classifier` rule scans the named skill for forbidden coupling content. |
| `source_team` | string or `null` or `"*"` | optional | If the prefix is fed by another team, name the source team. `null` means same-team or external producer. The literal `"*"` declares a **universal-source intake** — any team's members may write. See § Universal-source intakes. |

### Output entry

```jsonc
{
  "prefix": "audience-scan/*",
  "destination_kind": "knowledge",
  "destination_team": null,
  "destination_path": null,
  "schema": "audience-scan"
}
```

| Field | Type | Required | Meaning |
|---|---|---|---|
| `prefix` | string with optional `*` suffix | yes | Topic-prefix this member writes. |
| `destination_kind` | enum | yes | `knowledge` \| `decision` \| `por_file` \| `capability_gap` \| `skill_proposal` \| `backlog` |
| `destination_team` | string or `null` | optional | If the destination is on another team's surface, name it. |
| `destination_path` | string or `null` | optional | Required when `destination_kind` is `por_file`; names the path under `docs/`. |
| `schema` | string | optional | References a front-matter shape declared on the *producer's* taxonomy (`taxonomy.schemas.<id>`). The validator's `missing_destination_schema` warning fires when set but unresolvable. The producer's taxonomy owns the schema even when the consumer is on another team. |

### `destination_kind` semantics

| Kind | Meaning | Validation |
|---|---|---|
| `knowledge` | Output is a team knowledge entry under this prefix | No further validation |
| `decision` | Output is a decision context | Cross-checked in `orphan_output` rule |
| `por_file` | Output is a markdown edit (proposed via decision) under `docs/` | `dangling_por_sink` checks `destination_path` exists |
| `capability_gap` | Output is a `capability-gap` decision | `raises_capability_gaps` should be true |
| `skill_proposal` | Output is a proposal to author/modify a skill | Should be claimed by `skill-optimizer` |
| `backlog` | Output is a backlog item handed off | Cross-team consumption |

## Validation rules

Implemented in `scenarios/prompt-manager/api/memberflow/validation.go`. Run via `prompt-manager graph topics`.

| Rule | Smell | Severity | Detection |
|---|---|---|---|
| `orphan_output` | Knowledge output prefix has no consumer | warning | For each `destination_kind=knowledge` output: any member declares the prefix on `intake[]`, `required_read[]`, or `evidence_consumed[]`. Non-knowledge sinks (decision, por_file, capability_gap, skill_proposal, backlog) are never orphans. |
| `orphan_input` | Intake prefix has no producer | error | For each intake: another member's output prefix overlaps, or an `external_producer` claims it, or `source_team == "*"` |
| `unread_required` | `required_read[]` prefix has no producer | warning | For each required_read entry (excluding `source_team == "*"`): some member's `output[]` overlaps. Member-level `external_producers` is intentionally NOT honored here so the rule surfaces drift between declared read and write prefixes. |
| `conflicting_drain` | Two members' intake prefixes overlap | error | Pairwise overlap check |
| `wildcard_source_misuse` | `source_team == "*"` intake without any documented `external_producers` | warning | Universal-source declarations need a producer-side anchor (writer skill, external system) for auditability. |
| `unknown_taxonomy` | `intake[].taxonomy` does not resolve in the registry | error | Cross-check against `LoadAllTaxonomies` |
| `missing_taxonomy` | `intake[].taxonomy` is unset | error | Surfaces intake entries that have not been migrated to the inbox-flow taxonomy model. |
| `non_portable_classifier` | `intake[].classifier_skill` SKILL.md contains forbidden coupling content | error | Forbidden-pattern grep against the skill's SKILL.md |
| `missing_destination_schema` | `output[].schema` doesn't resolve under any taxonomy | warning | Cross-check against `taxonomy.schemas.<id>` across the registry |
| `dangling_por_sink` | `destination_kind=por_file` references missing `destination_path` | error | Filesystem stat |
| `dangling_evidence_decision` | `evidence_consumed[].for_decisions[]` references a decision-context id not declared in any team.json | error | Cross-check against `LoadAllTeamContracts` |
| `topic_key_prefix_mismatch` | Knowledge entry's `topic` doesn't match any declared prefix on its team | warning | Per-entry cross-check against the team's combined intake/output prefix set |
| `stalled_drain` | Intake has unrouted entries older than threshold (default 7d) | warning | Cross-check against `team knowledge-list` timestamps |
| `piling_inbox` | Intake has > N unrouted entries (default 50) | warning | Same query |

Errors fail `prompt-manager graph topics` with exit code 1. Warnings do not affect exit code.

`unread_required` is the producer-side mirror of `orphan_output`: when both fire on a related prefix pair (declared output `X/*` with no consumer, declared required_read `Y/*` with no producer, where X ≠ Y), the operator's reconciliation is a rename — pick one canonical prefix and align both sides. New declarations should land already-aligned; reconciling pre-existing drift is the explicit job of the topic-validation refactor's reconciliation phase.

## Prefix-match semantics

- Exact prefix `foo/bar` matches only `foo/bar`.
- Wildcard prefix `foo/bar/*` matches any topic starting with `foo/bar/`.
- Bare `*` is disallowed.
- Two prefixes overlap when either is a prefix of the other (with `/*` truncated). `research-inbox/audience/*` overlaps `research-inbox/*` but not `research-inbox/competitor/*`.

## Empty-file convention

```json
{}
```

is a positive "audited; no flow" declaration; absent files are treated the same way.

## Cross-team flow

When a member writes to another team's surface:

- Source side: `output[].destination_team = "<other-team>"` and `destination_kind` matches.
- Destination side: a member on the other team has an intake whose `prefix` matches, with `source_team` set to the producer team.

`orphan_output` flags one-sided claims.

### Producer owns the schema

When a prefix crosses team boundaries, the **producer's taxonomy** owns the front-matter schema. The consumer adopts the same schema; it does not redefine it. This rule is canonical and called out separately in `INTAKE_PIPELINE.md` ("Cross-team schema ownership").

### Universal-source intakes

`intake[].source_team = "*"` is a first-class declaration meaning *any team's members may write this prefix*. Used when an inbox is structurally cross-team — every agent on every team should be able to file into it without per-team plumbing.

Validator behavior:
- `orphan_input` is **skipped** for universal-source intakes (no specific peer-output is required to satisfy the producer-existence check).
- `wildcard_source_misuse` (warning) fires when `source_team == "*"` AND `external_producers` is empty. The wildcard declares "any team may write," but the producer-side anchor — typically the writer skill — must still be documented in `external_producers`. Empty `external_producers` + wildcard source means "I made it universal but forgot to document who actually writes."

When to use:
- The intake is genuinely universal — every agent on every team is an expected producer (e.g., bug reports, friction signals, capability gaps if cross-team filing is allowed). Two such flows exist today: `bug-inbox/*` on scenario-qa (drained by bug-investigator, fed by `report-bug`) and `friction-inbox/*` on meta-optimization (drained by friction-curator, fed by `report-friction`).
- A single writer skill is the producer-side anchor. The skill is destination-coupled by design (writer skills always are; the `non_portable_classifier` rule applies only to classifier skills, not writers).

When *not* to use:
- The set of producers is small and named — declare specific `source_team` values for each, not `*`.
- The intake is intra-team only — leave `source_team: null`.
- You don't yet have a writer skill — author the skill first; declare `source_team: "*"` and `external_producers: ["<skill-id>"]` together.

**Trigger guidance — where agents learn to invoke the writer.** The trigger paragraph that tells every agent when to load and invoke the writer skill is rendered into every member's heartbeat prompt as part of the Storage Map's `## Observe` subsection — see `scenarios/prompt-manager/api/heartbeat/prompt_builder.go` (`buildStorageMapSection`). When you add a new universal-source intake, add a matching paragraph there alongside the existing typed-topic-routing, bug-reporting (`report-bug`), and friction-reporting (`report-friction`) paragraphs so producers actually receive the trigger. The rendering is currently hardcoded prose for the two existing flows (bugs, friction); a third universal-source intake would push this past the worth-keeping-hardcoded threshold and into data-driven rendering off `intake[].source_team == "*"` declarations on member topics.json. Surface the refactor proposal as a `meta-self-improvement` decision when that third instance is in flight.

Worked example: `scenario-qa/bug-investigator` drains `bug-inbox/*`. Every team's members may file a bug via the `report-bug` writer skill. Topology declared as `intake[].source_team: "*"` + `external_producers: ["report-bug"]`. The investigator validates the producer's signal-type assignment as the first step of investigation; no separate classifier skill (deterministic-prefix routing).

Sister example: `meta-optimization/friction-curator` drains `friction-inbox/*` against the `friction-report` taxonomy. Every team's members may file friction via the `report-friction` writer skill. Topology declared identically: `intake[].source_team: "*"` + `external_producers: ["report-friction"]`. The curator validates scope (or reclassifies `unknown`), then routes by writing the entry to the appropriate `friction/<scope>/<date>/<slug>` topic owned by an existing meta-optimization sub-member. Critically, the curator owns no decision contexts — routing is determinate from scope; the destination scoped-topic owners (toolchain-validator, run-introspector, team-agent-optimizer, debt-curator) raise capability-gaps and other decisions after they drain the routed entries. This is the divergence from bug-investigator's pattern, which does own `bug-resolution-proposal` because investigation produces cross-cutting fixes; friction-curator produces routing only.

Worked example: `marketing-crew/researcher` writes `monetization-benchmark-adjacent-record/*` for the monetization team to consume. The schema for that prefix lives on the marketing-research taxonomy (`docs/marketing/signal-taxonomy.json#schemas.monetization-benchmark-adjacent`), not on `monetization-validation`. The validator's `missing_destination_schema` rule resolves `output[].schema` against the producer's taxonomy, not the consumer's. The consumer's `intake[].taxonomy` governs only routing/dispatch on the receiving side, not the on-disk shape of incoming entries.

## Example: marketing-crew researcher

```json
{
  "intake": [
    {
      "prefix": "research-inbox/*",
      "taxonomy": "marketing-research",
      "classifier_skill": "marketing-signal-classifier",
      "source_team": null
    }
  ],
  "output": [
    { "prefix": "audience-scan/*",                  "destination_kind": "knowledge", "destination_team": null,           "schema": "audience-scan" },
    { "prefix": "competitor-record/*",                     "destination_kind": "knowledge", "destination_team": null,           "schema": "competitor-observation" },
    { "prefix": "hook-record/*",                           "destination_kind": "knowledge", "destination_team": null,           "schema": "hook" },
    { "prefix": "monetization-benchmark-adjacent-record/*", "destination_kind": "knowledge", "destination_team": "monetization", "schema": "monetization-benchmark-adjacent" }
  ],
  "decisions_owned": ["audience-update", "channel-strategy-update", "post-type-proposal", "hook-candidate-promotion"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["vision-walk", "operator", "bookmark-intelligence-hub"]
}
```

When loaded, `prompt-manager graph topics --team marketing-crew` should:
- Render edges from `vision-walk` and `operator` (external boundary nodes) into the researcher.
- Render edges from the researcher to four output prefixes (three same-team knowledge sinks, one cross-team monetization sink).
- Validate that `marketing-research` taxonomy exists and resolves to `docs/marketing/signal-taxonomy.json`.
- Validate that `marketing-signal-classifier` is a registered, portable skill (no forbidden coupling content).
- Validate every `output[].schema` resolves against the producer's taxonomy.
- Cross-validate `monetization-benchmark-adjacent-record/*` against the monetization team's intake (some monetization member should declare this prefix in their `intake` with `source_team: "marketing-crew"`).

## Stability gate

This schema is canon as of the inbox-flow refactor (Phase I cleanup landed; five adopters in production: `marketing-crew/researcher`, `marketing-crew/brand-manager`, `meta-optimization/debt-curator`, `monetization/opportunity-scout`, `monetization/market-validator`).

Backwards-incompatible changes from here require a `meta-optimization` decision and a migration plan covering: (a) every `topics.json` file in `scenarios/prompt-manager/store/teams/*/members/*/`, (b) the Go schema at `scenarios/prompt-manager/api/memberflow/schema.go`, (c) the heartbeat builder section template at `scenarios/prompt-manager/api/heartbeat/inbox_flow.go`, and (d) the validation rules.

Backwards-compatible additions (new optional fields, new `destination_kind` enum values, new validation rules at `warning` severity) may land via PR without a decision.
