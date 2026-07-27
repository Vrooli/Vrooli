# `topics.json` — per-member message-flow declarations

**Status:** canon. This document is the plan-of-record for the `topics.json` data layer — Pillar 1 of [topic validation](PRIMITIVES.md#three-pillars-of-topic-validation). It pairs with the Go implementation at `path:scenarios/prompt-manager/api/memberflow/schema.go`. Cross-team-readable; cited by `INTAKE_PIPELINE.md`, `LAYERS.md`, and the heartbeat builder. Sibling pillars: [`PROSE_SCAN_TARGETS.md`](PROSE_SCAN_TARGETS.md) (P2 — prose scan) and [`RUNTIME_ATTRIBUTION.md`](RUNTIME_ATTRIBUTION.md) (P3 — observed writes).

Promoted from `drafts/topics-schema.md` after the inbox-flow refactor stabilized the schema across five in-prod adopters. Backwards-incompatible changes from this point require a `meta-optimization` decision and a migration plan (see [Stability gate](#stability-gate)).

## Purpose

A `topics.json` file declares, **structurally**, how a single team member produces and consumes work via topic-prefixed channels. Team-level `team.json::topicCatalog` declares shared topic-family status and purpose. Together, those structured declarations are the machine-readable substrate the heartbeat builder uses to render generated prompt sections, and the validator uses to detect orphan flows and drift.

Once this layer ships, prose claims like "the researcher drains `research-inbox/*`" become declarations the system can validate, visualize, and lint against. Orphan output prefixes, dangling intake claims, conflicting drain duty, and stalled inboxes become detectable.

`topics.json` is **per-member**, not per-team or per-skill. The topic declarations live at the same granularity as `RESPONSIBILITIES.md` and `HEARTBEAT.md`. Shared topic-family metadata belongs on the team, not duplicated in every member file.

## File location

```
scenarios/prompt-manager/store/teams/<team>/members/<member>/topics.json
```

Sibling to `HEARTBEAT.md`, `RESPONSIBILITIES.md`, `last-handoff.md`. One file per member.

Team-level topic-family metadata lives in:

```
scenarios/prompt-manager/store/teams/<team>/team.json::topicCatalog
```

## Loop kinds

Every member declares `loop_kind`. One question generates the taxonomy and the obligation it carries: **where does this loop keep its memory between heartbeats?**

| Loop kind | Memory lives in | Must declare a ledger |
|---|---|---|
| `queue` | the intake's unrouted set — draining an entry is what marks it done | no |
| `reactive` | the pending-artifact set it reviews, typically pending decisions | no |
| `sweep` | nowhere by default | **yes** |
| `generative` | nothing; the loop produces on a cadence over no population | no |

A sweep iterates a standing population that never empties — every skill, every agent, every scenario. Nothing marks a target done, so without memory the member re-picks the head of its priority ladder every heartbeat and never reaches the tail. That is the whole reason the field exists.

`generative` is a positive declaration ("nothing to remember"), not a null. An omitted `loop_kind` means undeclared, which is a different and reportable state.

### Coverage ledger

A sweep's memory is a **coverage ledger**: one topic prefix that the member both writes and reads back. The naming shape is `<subject>-visited/<id>`, registered in [`TOPICS.md`](TOPICS.md) § Topic families. One entry per target, superseded on re-visit — current state, not an event log.

Self-reference is the defining property, not an accident: the writer reads it because the next heartbeat's target selection depends on what the last one reached. This is why a ledger legitimately satisfies `orphan_output` without a peer consumer.

A member may sweep more than one population and carry a ledger per population; `team-agent-optimizer` sweeps teams and agents on one ladder. By convention the subject in each ledger prefix names its population, which is how coverage is attributed per population rather than pooled.

The ladder and the stop condition are **not** structural. A ledger gives the loop memory; the priority ladder that consumes it and the quiet-period rule that lets a completed rotation rest both live in the member's `HEARTBEAT.md` as prose. Validation does not check them, and canon should not pretend it does — a sweep with a ledger and no stop condition still converges more than one without a ledger, but it does not rest.

**A self-read output is not by itself a ledger.** A member re-reading its own durable record for continuity — `outcome-target-record/*`, `vision-walk-record/*` — is a third valid shape. What distinguishes a ledger is purpose: it exists to inform target selection over a population.

## Schema (canonical)

```json
{
  "loop_kind": "sweep",
  "population": ["skill", "action"],
  "intake": [
    {
      "prefix": "research-inbox/*",
      "taxonomy": "marketing-research",
      "classifier_skill": "signal-classifier",
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

## Team topic catalog

`team.json::topicCatalog` is the structured source of truth for topic-family status and purpose. It does not declare readers or writers; per-member `topics.json` files remain the source of runtime member relationships.

```jsonc
{
  "topicCatalog": [
    {
      "prefix": "audience-scan/*",
      "status": "live",
      "purpose": "Audience pain, vocabulary, buyer triggers, objections, and persona evidence."
    },
    {
      "prefix": "publish-performance/*",
      "qualifier": "future",
      "status": "target",
      "purpose": "Telemetry and qualitative performance."
    }
  ]
}
```

| Field | Type | Required | Meaning |
|---|---|---|---|
| `prefix` | string with optional `/*` suffix | yes | Topic family prefix. Uses the same shape rules as member topic prefixes. |
| `qualifier` | string | optional | Empty for current/live topics, `future` for target-state topics, `old` for historical topics, `external` for outside surfaces. When omitted, the validator derives the expected qualifier from `status`. |
| `status` | enum | yes | One of `live`, `live transitional`, `live system`, `live but under-consumed`, `target`, `old`, or `external`. |
| `purpose` | string | required for current/live statuses | Canonical team-level description rendered into generated `# Topic Contract` prompt sections and validated against operating-model Topic Catalog tables. |

The operating-model Topic Catalog table is a human-readable projection of this structured metadata plus the graph/runtime relationships. Purpose text in docs must match `team.json::topicCatalog`; owner/reader cells must match graph and member `topics.json` relationships.

### Intake entry

```jsonc
{
  "prefix": "research-inbox/*",
  "taxonomy": "marketing-research",
  "classifier_skill": "signal-classifier",
  "source_team": null
}
```

| Field | Type | Required | Meaning |
|---|---|---|---|
| `prefix` | string with optional `*` suffix | yes | Topic-prefix matched against `team knowledge-list --topic-prefix=`. |
| `taxonomy` | string | yes | The id of the taxonomy JSON sidecar (`path:docs/<domain>/<id>.json`) that owns this prefix's signal vocabulary, dispatch, evidence rules, and destination schemas. The validator's `unknown_taxonomy` rule resolves this against the registry; an empty value fires `missing_taxonomy`. |
| `classifier_skill` | string | optional | A portable, member-agnostic judgment skill that assigns `signal_type` from the taxonomy. Optional: when the topic-prefix carries a deterministic signal type, no classifier is needed. The `prose_topic_leak` rule scans the named skill's SKILL.md for forbidden topic-coupling content (see `PROSE_SCAN_TARGETS.md`); the retired `non_portable_classifier` rule is subsumed by it. |
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

Implemented in `path:scenarios/prompt-manager/api/memberflow/validation.go`. Run via `prompt-manager graph topics`.

| Rule | Smell | Severity | Detection |
|---|---|---|---|
| `loop_kind_missing` | Member declares no loop kind | warning | Empty `loop_kind` on a member with any flow declaration. Warning during adoption: the value is a judgment call owned by the member's team, and inferring it fleet-wide would make the declaration layer fiction. Promote to error once every member carries a value. |
| `loop_kind_invalid` | Loop kind is not a known value | error | Not one of `queue` \| `reactive` \| `sweep` \| `generative`. |
| `loop_kind_intake_mismatch` | Member drains an intake but is not `queue` | error | `len(intake) > 0` and `loop_kind != queue`. This is what makes the field self-checking against declarations that already exist, so `loop_kind` cannot silently drift from the flow it describes. |
| `sweep_without_ledger` | A sweep has no coverage memory | error | `loop_kind == sweep` and no output prefix that the same member also reads. Gated behind the declaration, so it cannot fire on an undeclared member. |
| `ledger_shape_invalid` | Ledger-shaped output nobody reads back | error | An output matching `<subject>-visited/` not covered by its writer's own read declarations. Independent of `loop_kind` so a mis-shaped ledger is caught even before the loop kind is declared. |
| `sweep_population_missing` | A sweep does not name what it iterates | warning | `loop_kind == sweep` and `population` empty. Never hardens to an error: a population that is not graph-resident (a time window over recent runs) legitimately omits it and is honestly unmeasurable rather than silently counted as covered. |
| `orphan_output` | Knowledge output prefix has no consumer | warning | For each `destination_kind=knowledge` output: any member declares the prefix on `intake[]`, `required_read[]`, or `evidence_consumed[]`. Non-knowledge sinks (decision, por_file, capability_gap, skill_proposal, backlog) are never orphans. |
| `orphan_input` | Intake prefix has no producer | error | For each intake: another member's output prefix overlaps, or an `external_producer` claims it, or `source_team == "*"` |
| `unread_required` | `required_read[]` prefix has no producer | error | For each required_read entry (excluding `source_team == "*"`): some member's `output[]` overlaps, or a writer skill's `skill.json::writes_to[]` overlaps. Member-level `external_producers` is intentionally NOT honored here so the rule surfaces drift between declared read and write prefixes. |
| `conflicting_drain` | Two members' intake prefixes overlap | error | Pairwise overlap check |
| `wildcard_source_misuse` | `source_team == "*"` intake without any documented `external_producers` | warning | Universal-source declarations need a producer-side anchor (writer skill, external system) for auditability. |
| `unknown_taxonomy` | `intake[].taxonomy` does not resolve in the registry | error | Cross-check against `LoadAllTaxonomies` |
| `missing_taxonomy` | `intake[].taxonomy` is unset | error | Surfaces intake entries that have not been migrated to the inbox-flow taxonomy model. |
| `missing_destination_schema` | `output[].schema` doesn't resolve under any taxonomy | warning | Cross-check against `taxonomy.schemas.<id>` across the registry |
| `dangling_por_sink` | `destination_kind=por_file` references missing `destination_path` | error | Filesystem stat |
| `dangling_evidence_decision` | `evidence_consumed[].for_decisions[]` references a decision-context id not declared in any team.json | error | Cross-check against `LoadAllTeamContracts` |
| `team_role_member_drift` | `roles.json` and the contract member set disagree | error | Both directions: a role id with no matching `team.json::operatingContract.members` entry, and a member with no role entry. Teams that declare no `roles.json` are skipped — the file is optional, and requiring one from every team is a separate decision. |
| `topic_key_prefix_mismatch` | Knowledge entry's `topic` doesn't match any declared prefix on its team | warning | Cross-check against the team's combined intake/output prefix set, grouped per topic family (see § Per-defect grouping) |
| `stalled_drain` | Intake has unrouted entries older than threshold (default 7d) | warning | Cross-check against `team knowledge-list` timestamps |
| `piling_inbox` | Intake has > N unrouted entries (default 50) | warning | Same query |
| `member_doc_file_missing` | Member has no `HEARTBEAT.md` / `RESPONSIBILITIES.md` | error / warning | Filesystem stat. Error for `HEARTBEAT.md` — that member runs the generic fallback task. Warning for `RESPONSIBILITIES.md` — the member still receives its full generated contract. |
| `member_doc_section_missing` | Required canonical section absent | error | Level-two heading scan, fenced blocks skipped. Required set is `TEAM_MEMBER_ARCHITECTURE.md` §"Canonical section vocabulary". |
| `member_doc_section_alias` | Retired heading name in use | error | Heading matches the retired-alias table. The alias satisfies its canonical slot, so a rename reports once rather than as both an alias and a missing section. |
| `member_doc_section_duplicate` | One canonical section declared twice | error | Counts canonical headings plus aliases mapped onto them, so a file mid-rename reports the rename *and* the resulting duplicate. |
| `member_doc_section_recommended` | Recommended section absent | warning | Never hardens to an error: the content is team judgment, and an error would push a validator into authoring prose. Closes through the owning team's decision context. |

Errors fail `prompt-manager graph topics` with exit code 1. Warnings do not affect exit code.

The `member_doc_*` family checks the two per-member Markdown files that reach a running agent verbatim. The vocabulary they enforce is canon in `TEAM_MEMBER_ARCHITECTURE.md` §"Member document contract"; the tables in `path:scenarios/prompt-manager/api/memberflow/member_doc.go` are its machine copy. Fenced-block skipping is load-bearing rather than defensive — every `HEARTBEAT.md` embeds a handoff template whose first line is `## HANDOFF`, and a naive scan reports it as a section on every member.

### Per-defect grouping

Rules that scan `knowledge.jsonl` — `actual_writer_undeclared` and `topic_key_prefix_mismatch` — report **one finding per (member, topic family)**, not one per entry.

An undeclared prefix fires on every entry the member ever wrote under it, so a per-entry count measures how long the team has been running, not how many declarations need fixing. One missing `output[]` line produced 25 errors; the repair was one line. Reported that way, a sweep reads as a large repair job and the operator cannot see how many distinct declarations actually need attention.

The collapsed finding carries the family prefix, the occurrence count, and a sample entry id, so the evidence trail survives. The family is derived by truncating the observed topic at its first variable segment — an ISO date, a generated id, or a bare number — which yields the prefix an operator would declare:

| Observed topic | Family |
|---|---|
| `initiative-portfolio-record/2026-05-14` | `initiative-portfolio-record/*` |
| `challenge-resolution-record/dec-1778803361775636366` | `challenge-resolution-record/*` |
| `friction-report/recurring-workaround/2026-06-15/toolchain-fallback` | `friction-report/recurring-workaround/*` |
| `contrarian-scan-2026-06-14` | `contrarian-scan-*` |
| `quality-audit/audio-tools/api-steer` | unchanged — no variable segment |

The fourth row is the case that shaped the rule. A member that builds its topic key by string concatenation instead of path join produces a malformed family, and collapsing it to `contrarian-scan-*` makes a month of the same typo read as one defect — while keeping it visibly distinct from the correctly-formed `contrarian-scan/*` family, because a malformed key and a missing declaration are two different problems with two different fixes.

Rules whose per-entry shape is deliberate keep it. The external-write threshold exists to name exactly which entries crossed the cap, so it is not grouped.

`unread_required` is the producer-side mirror of `orphan_output`: when both fire on a related prefix pair (declared output `X/*` with no consumer, declared required_read `Y/*` with no producer, where X ≠ Y), the operator's reconciliation is a rename — pick one canonical prefix and align both sides. New declarations should land already-aligned; reconciling pre-existing drift is the explicit job of the topic-validation refactor's reconciliation phase.

The rule consults writer-skill `writes_to[]` as a producer source, not only member `output[]`. A required_read prefix that overlaps a writer skill's `writes_to[]` has a documented producer; demanding a member-side `output[]` in addition would force false declarations (e.g., friction-curator does not write `friction-inbox/*` — the `report-friction` skill does).

## Findings reach the member that caused them

Validation output has two audiences with different needs, and for a long time it served only one. `prompt-manager graph topics` is an operator sweep: someone runs it, reads everything, and decides. The member whose declarations are wrong was never told, so a defect could repeat on every heartbeat indefinitely — one member wrote its snapshot topic with a hyphen where the prefix needed a path separator on twenty consecutive runs, every one of them validated, none of them reported to the agent.

The `# Contract Findings` heartbeat prompt section closes that loop (`path:scenarios/prompt-manager/api/heartbeat/contract_findings.go`). Three rules govern it:

1. **Attribution decides inclusion, not an allow-list.** A finding appears in a member's prompt when `Finding.Member` names that member. Any rule that attributes to a member reaches that member automatically; adding a rule does not mean editing the prompt builder. Findings with no member attribution belong to the team or the corpus and are not routed to anyone — choosing a recipient would be a guess.
2. **Advisory findings are withheld.** See `PROSE_SCAN_TARGETS.md` §"Advisory findings". A heuristic that cannot separate a real defect from a lookalike is review material, not an instruction.
3. **A clean member sees nothing.** The section is omitted entirely when there are no findings, so the loop costs no prompt budget on a healthy fleet. An unwired provider also renders nothing rather than claiming a clean contract it never checked — an unwired builder and a clean member must not look alike to a reader of the prompt.

The section states the routes a member can actually take: correct its own writes this run, propose the declaration change as a decision in an owned context, or report it as friction. Members do not edit `topics.json` or their own member contract directly.

## Prefix-match semantics

- Example exact prefix `topic[example]:foo/bar` matches only the value `literal:foo/bar`.
- Example wildcard prefix `topic[example]:foo/bar/*` matches any topic starting with `literal:foo/bar/`.
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
- A single writer skill is the producer-side anchor. The skill is destination-coupled by design (writer skills always are; the `prose_topic_leak` classifier-purity check applies only to classifier skills, not writers).

When *not* to use:
- The set of producers is small and named — declare specific `source_team` values for each, not `*`.
- The intake is intra-team only — leave `source_team: null`.
- You don't yet have a writer skill — author the skill first; declare `source_team: "*"` and `external_producers: ["<skill-id>"]` together.

### Evidence-consumed sidecars

Some topics are not drained as inboxes but still must be visible to members when they own a related decision. Declare those with `evidence_consumed[]`, not `intake[]`.

The canonical example is contrarian review:

- a contrarian writes `challenge-report/<decision-id>` as append-only evidence;
- the same contrarian writes `challenge-resolution-record/<decision-id>` as the latest state;
- the challenged decision's author consumes both prefixes as evidence for its owned decision contexts;
- the author does not drain or delete the entries.

This distinction matters for validation and prompt generation. `orphan_output` treats `evidence_consumed[]` as a real consumer because the topic has a reader, even though no router skill drains it. The lifecycle is documented in [`CONTRARIAN_REVIEW.md`](CONTRARIAN_REVIEW.md).

**Trigger guidance — where agents learn to invoke the writer.** The trigger paragraph that tells every agent when to load and invoke the writer skill is rendered into every member's heartbeat prompt as part of the Storage Map's `## Observe` subsection — see `path:scenarios/prompt-manager/api/heartbeat/prompt_builder.go` (`buildStorageMapSection`). When you add a new universal-source intake, add a matching paragraph there alongside the existing typed-topic-routing, bug-reporting (`report-bug`), and friction-reporting (`report-friction`) paragraphs so producers actually receive the trigger. The rendering is currently hardcoded prose for the two existing flows (bugs, friction); a third universal-source intake would push this past the worth-keeping-hardcoded threshold and into data-driven rendering off `intake[].source_team == "*"` declarations on member topics.json. Surface the refactor proposal as a `meta-self-improvement` decision when that third instance is in flight.

Worked example: `team:scenario-qa/bug-investigator` drains `bug-inbox/*`. Every team's members may file a bug via the `report-bug` writer skill. Topology declared as `intake[].source_team: "*"` + `external_producers: ["report-bug"]`. The investigator validates the producer's signal-type assignment as the first step of investigation; no separate classifier skill (deterministic-prefix routing).

Sister example: `team:meta-optimization/friction-curator` drains `friction-inbox/*` against the `friction-report` taxonomy. Every team's members may file friction via the `report-friction` writer skill. Topology declared identically: `intake[].source_team: "*"` + `external_producers: ["report-friction"]`. The curator validates scope (or reclassifies `unknown`), then routes by writing the entry to the appropriate `friction-report/<scope>/<date>/<slug>` topic owned by an existing meta-optimization sub-member. Critically, the curator owns no decision contexts — routing is determinate from scope; the destination scoped-topic owners (toolchain-validator, run-introspector, team-agent-optimizer, debt-curator) raise capability-gaps and other decisions after they drain the routed entries. This is the divergence from bug-investigator's pattern, which does own `bug-resolution-proposal` because investigation produces cross-cutting fixes; friction-curator produces routing only.

The producer-owns-schema rule has its own worked example (marketing → monetization) in `INTAKE_PIPELINE.md` § Cross-team schema ownership; see § Producer owns the schema above for the one-line statement and cite.

## Example: marketing-crew researcher

```json
{
  "intake": [
    {
      "prefix": "research-inbox/*",
      "taxonomy": "marketing-research",
      "classifier_skill": "signal-classifier",
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
- Validate that `marketing-research` taxonomy exists and resolves to `path:docs/marketing/taxonomies/marketing-research/taxonomy.json`.
- Validate that `signal-classifier` is a registered, portable skill (no forbidden coupling content).
- Validate every `output[].schema` resolves against the producer's taxonomy.
- Cross-validate `monetization-benchmark-adjacent-record/*` against the monetization team's intake (some monetization member should declare this prefix in their `intake` with `source_team: "marketing-crew"`).

## Stability gate

This schema is canon as of the inbox-flow refactor (Phase I cleanup landed; five adopters in production: `team:marketing-crew/researcher`, `team:marketing-crew/brand-manager`, `team:meta-optimization/debt-curator`, `team:monetization/opportunity-scout`, `team:monetization/market-validator`).

Backwards-incompatible changes from here require a `meta-optimization` decision and a migration plan covering: (a) every `topics.json` file in `path:scenarios/prompt-manager/store/teams/*/members/*/`, (b) every `team.json::topicCatalog` that describes those topic families, (c) the Go schemas at `path:scenarios/prompt-manager/api/memberflow/schema.go` and `path:scenarios/prompt-manager/api/memberflow/team_contracts.go`, (d) the heartbeat builder section templates, and (e) the validation rules.

Backwards-compatible additions (new optional fields, new `destination_kind` enum values, new validation rules at `warning` severity) may land via PR without a decision.
