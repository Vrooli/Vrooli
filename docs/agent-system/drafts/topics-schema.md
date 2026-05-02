# `topics.json` — per-member message-flow declarations

This document is the human-readable plan-of-record for the `topics.json` data layer. It pairs with the Go implementation at `scenarios/prompt-manager/api/memberflow/schema.go`. Until the schema is approved by `meta-optimization` decision (the Phase 0 acceptance gate), this lives under `drafts/`. After acceptance it is promoted to `docs/agent-system/TOPICS_SCHEMA.md`.

## Purpose

A `topics.json` file declares, **structurally**, how a single team member produces and consumes work via topic-prefixed channels. It is the machine-readable counterpart to prose router skills.

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
      "drained_by_skill": "marketing-research-router",
      "source_team": null
    }
  ],
  "output": [
    {
      "prefix": "audience-scan/*",
      "destination_kind": "knowledge",
      "destination_team": null
    },
    {
      "prefix": "monetization-benchmark-adjacent/*",
      "destination_kind": "knowledge",
      "destination_team": "monetization"
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
| `intake` | array of intake entries | Topic-prefixes this member drains. Each entry names a router/method skill that knows how to drain it. |
| `output` | array of output entries | Topic-prefixes this member writes to. Each entry names what the destination is and where. |
| `decisions_owned` | array of decision-context strings | Decision contexts this member owns (i.e., proposes-and-files, not just consumes). |
| `decisions_consumed` | array of decision-context strings | Decision contexts whose acceptance changes this member's behavior or workload. |
| `raises_capability_gaps` | boolean | True if this member's role includes filing `capability-gap` decisions when blocked by missing tooling. |
| `external_producers` | array of stable identifiers | Non-team-member producers that feed this member's intake (e.g., `vision-walk`, `operator`, `bookmark-intelligence-hub`). Used to satisfy the `orphan_input` validation rule for prefixes whose producer is not another team member. |

### Intake entry

```jsonc
{
  "prefix": "research-inbox/*",
  "drained_by_skill": "marketing-research-router",
  "source_team": null
}
```

| Field | Type | Required | Meaning |
|---|---|---|---|
| `prefix` | string with optional `*` suffix | yes | Topic-prefix matched against `team knowledge-list --topic-prefix=`. Must end in `/*` for a prefix-match; an exact-prefix string (no `*`) matches only that exact topic. |
| `drained_by_skill` | string | yes | Skill ID that owns the drain procedure for this prefix. The validation rule `missing_drain_skill` ensures this skill exists. |
| `source_team` | string or `null` | optional | If the prefix is fed by another team's members (cross-team flow), name the source team here. `null` means same-team or external producer. Used by the `orphan_input` rule to find the producer. |

### Output entry

```jsonc
{
  "prefix": "audience-scan/*",
  "destination_kind": "knowledge",
  "destination_team": null,
  "destination_path": null
}
```

| Field | Type | Required | Meaning |
|---|---|---|---|
| `prefix` | string with optional `*` suffix | yes | Topic-prefix this member writes. |
| `destination_kind` | enum | yes | `knowledge` \| `decision` \| `por_file` \| `capability_gap` \| `skill_proposal` \| `backlog` |
| `destination_team` | string or `null` | optional | If the destination is on another team's surface (e.g., monetization-team knowledge), name it. `null` means same-team. |
| `destination_path` | string or `null` | optional | Required when `destination_kind` is `por_file`; names the path under `docs/` to which content is promoted. |

### `destination_kind` semantics

| Kind | Meaning | Validation |
|---|---|---|
| `knowledge` | Output is a team knowledge entry under this prefix | No further validation |
| `decision` | Output is a decision context; its name should appear in some member's `decisions_consumed` | Cross-checked in `orphan_output` rule |
| `por_file` | Output is a markdown edit (proposed via decision) to a file under `docs/` | `dangling_por_sink` rule checks `destination_path` exists |
| `capability_gap` | Output is a `capability-gap` decision | Member's `raises_capability_gaps` should be true |
| `skill_proposal` | Output is a proposal to author/modify a skill | Should be claimed by `skill-optimizer` (validated by cross-team consumption) |
| `backlog` | Output is a backlog item handed off to a director-swarm or scenario team | Cross-team consumption (`source_team` of the consuming member's intake should reference this team) |

## Validation rules

Implemented as a Go package (`scenarios/prompt-manager/api/memberflow/validation.go`) with one function per rule. Run via `prompt-manager graph topics`.

| Rule | Smell | Severity | Detection |
|---|---|---|---|
| `orphan_output` | Output prefix has no consumer (no router intake matches, no PoR sink declared) | error | For each output entry: at least one member's intake `prefix` must overlap, OR `destination_kind` resolves to a non-member sink |
| `orphan_input` | Intake prefix has no producer | error | For each intake entry: at least one member's output `prefix` must overlap, OR an `external_producer` claims it |
| `conflicting_drain` | Two members' intake prefixes overlap (both claim to drain the same topic) | error | Pairwise overlap check across all members; equal prefixes always conflict, prefix-with-`*` overlaps any narrower prefix |
| `missing_drain_skill` | `drained_by_skill` references a skill ID that doesn't exist | error | Cross-check against `prompt-manager skill list --json` |
| `dangling_por_sink` | `destination_kind=por_file` references a `destination_path` that doesn't exist | error | Filesystem stat |
| `stalled_drain` | Intake prefix has unrouted entries older than threshold (default 7 days) | warning | Cross-check against `prompt-manager team knowledge-list --topic-prefix=` timestamps |
| `piling_inbox` | Intake prefix has > N unrouted entries (default 50) | warning | Same query |

Errors fail `prompt-manager graph topics` with exit code 1. Warnings do not affect exit code.

## Prefix-match semantics

- An exact prefix string `foo/bar` matches only the topic `foo/bar` (used for fixed topics like `audience-update` decision contexts when expressed as topic strings, though in practice decisions are not topics — this would be unusual).
- A wildcard prefix `foo/bar/*` matches any topic starting with `foo/bar/` (most common form).
- A wildcard `*` matches everything (disallowed in practice; would defeat overlap detection).
- Two prefixes "overlap" if either is a prefix of the other (with `/*` truncated for comparison). For example, `research-inbox/audience/*` overlaps `research-inbox/*` but does not overlap `research-inbox/competitor/*`.

The `conflicting_drain` rule treats overlap as conflict only when the overlap is non-empty in practice (i.e., at least one prefix is "wider" than the other). Equal prefixes always conflict.

## Empty-file convention

A member with no flow declarations writes:

```json
{}
```

This is valid. The validation engine treats absent keys as empty arrays. Members with no flow may omit the file entirely, but explicit `{}` makes "we audited this and it has no flow" a positive declaration rather than ambiguous absence.

## Cross-team flow

When a member writes to another team's surface, both sides should agree:

- Source side: `output.destination_team = "<other-team>"` and `destination_kind` matches
- Destination side: a member of the other team has an intake entry whose `prefix` matches the source's `prefix`, with `source_team` set to the originating team

Validation flags one-sided claims via `orphan_output` (the source claims a destination team that doesn't pick it up) or `orphan_input` (the destination claims a source team that doesn't write it).

## Backwards compatibility

None. This is greenfield. Members without `topics.json` are listed by `prompt-manager graph topics` under "members with no declarations" and the structural layer scores in the audit skill (Phase 5) report `0 missing` for the four pipeline layers.

## Example: marketing-crew researcher

The canonical worked example (used to validate the schema during Phase 2 canary backfill):

```json
{
  "intake": [
    {
      "prefix": "research-inbox/*",
      "drained_by_skill": "marketing-research-router",
      "source_team": null
    }
  ],
  "output": [
    { "prefix": "audience-scan/*", "destination_kind": "knowledge", "destination_team": null },
    { "prefix": "competitor/*", "destination_kind": "knowledge", "destination_team": null },
    { "prefix": "hook/*", "destination_kind": "knowledge", "destination_team": null },
    { "prefix": "monetization-benchmark-adjacent/*", "destination_kind": "knowledge", "destination_team": "monetization" }
  ],
  "decisions_owned": ["audience-update", "channel-strategy-update", "post-type-proposal", "hook-candidate-promotion"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["vision-walk", "operator", "bookmark-intelligence-hub"]
}
```

When loaded, `prompt-manager graph topics --team marketing-crew` should:
- Render edges from `vision-walk` and `operator` (external boundary nodes) into the researcher
- Render edges from the researcher to four output prefixes (three same-team knowledge sinks, one cross-team monetization sink)
- Validate that `marketing-research-router` skill exists
- Validate that `audience-update`, `channel-strategy-update`, `post-type-proposal`, `hook-candidate-promotion` are decision contexts at least one team member consumes (or, more loosely, that they exist)
- Cross-validate `monetization-benchmark-adjacent/*` against the monetization team's intake (some monetization member should declare this prefix in their `intake` with `source_team: "marketing-crew"`)

## Stability gate

After Phase 2 canary backfill on marketing-crew completes cleanly, this schema is frozen by a `meta-optimization` decision. Backwards-incompatible changes after that gate require a new decision and a migration plan for already-authored `topics.json` files.
