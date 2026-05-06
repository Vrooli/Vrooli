# Operating Graph Contracts

**Status:** canon. Operating graphs are the plan-of-record diagram layer for prompt-manager team workflow contracts. They connect human-readable Mermaid diagrams to the runtime declarations in `topics.json`, `team.json`, and generated heartbeat prompts.

This document extends `TOPICS_SCHEMA.md`. `topics.json` remains the runtime source for member topic flow. Operating graphs make the team-level shape visible and checkable.

## Purpose

Operating graphs let humans and agents design a team in Mermaid while validators prove the diagram still matches the machine-readable contract.

The intended flow is:

```text
PoR Mermaid diagram
  -> normalized operating graph
  -> validation against team.json and topics.json
  -> generated member prompt contracts
  -> prompt preview checks
```

Mermaid is not executed at runtime. Runtime prompts are generated from structured declarations.

## Validate vs Diff

Operating graph tooling has two related jobs:

| Command | Purpose | Output |
|---|---|---|
| `prompt-manager graph operating-model validate` | Gate whether the graph is a valid contract. | Severity-bearing findings: errors and warnings. |
| `prompt-manager graph operating-model diff` | Reconcile graph and runtime declarations. | Repair map grouped by drift direction. |

Keep these separate. Validation answers "can this be trusted as a contract?" Diff answers "what needs to change so the plan-of-record graph and runtime config say the same thing?"

Diff compares both directions:

| Direction | Meaning | Typical fix |
|---|---|---|
| `graph_relationship_missing_in_runtime` | Mermaid declares a relationship that `topics.json` / `team.json` do not back. | Add the matching runtime declaration, or remove the graph edge. |
| `runtime_relationship_missing_in_graph` | Runtime config declares a relationship absent from the Mermaid contract graph. | Add the graph edge, or remove the obsolete runtime declaration. |

For human output, diff groups these as "Graph Declares, Runtime Missing" and "Runtime Declares, Graph Missing." JSON output includes machine fields such as `relationship`, `source_path`, `line`, `runtime_path`, `acceptable_fields`, and `suggestions`.

## Metadata Block

Every checkable operating graph uses a metadata comment immediately followed by a Mermaid fence:

````markdown
<!-- prompt-manager-graph:
id: marketing-operating-model
scope: team
team: marketing-crew
mode: contract
-->
```mermaid
flowchart LR
  %% @node R member:researcher
  R[Researcher]
  %% @node RI topic:research-inbox/*
  RI[research-inbox/*]
  RI --> R
```
````

Required fields:

| Field | Meaning |
|---|---|
| `id` | Stable graph id used by CLI/API filters. |
| `scope` | Initial value is `team`. |
| `team` | Prompt-manager team id. |
| `mode` | `explanatory`, `checkable`, or `contract`. |

Optional fields:

| Field | Meaning |
|---|---|
| `status` | Human status such as `draft`, `target`, or `canon`. |
| `allow` | Comma-separated rule exceptions. Keep rare and justified in prose. |

## Modes

| Mode | Meaning | Validation |
|---|---|---|
| `explanatory` | Conceptual diagram. | Parse only. No contract checks. |
| `checkable` | Partial diagram with typed nodes. | Validate typed entities and direct edges that are present. Missing runtime relationships are allowed. |
| `contract` | Team operating contract. | Validate typed entities, direct edges, and selected completeness rules against runtime config. |

Use `contract` for the canonical team operating model. Use `checkable` for subset diagrams. Use `explanatory` for broad system views with summary nodes.

## Mermaid Subset

Only this subset is supported for operating graphs:

- `flowchart LR`, `flowchart TB`, `flowchart RL`, or `flowchart BT`
- Mermaid comments beginning with `%%`
- typed node annotations: `%% @node ID machine-token`
- node declarations: `ID[label]`, `ID["label"]`, `ID["machine-token<br/>Display"]`
- simple directed edges: `A --> B`
- optional edge labels: `A -->|label| B`; labels are parsed but ignored in the first validator

Unsupported syntax fails `contract` graphs. Do not use chained fanout, subgraphs, styling, classes, HTML tables, or implicit semantic labels in contract diagrams.

## Typed Nodes

Prefer invisible typed annotations so rendered diagrams stay readable:

```mermaid
%% @node R member:researcher
R[Researcher]
```

Inline typed labels are also supported for compact diagrams:

```mermaid
R["member:researcher<br/>Researcher"]
```

Do not combine both forms on the same node. The validator treats that as an ambiguous contract.

| Kind | Example | Validation source |
|---|---|---|
| `member` | `member:researcher` | `team.json::operatingContract.members` |
| `topic` | `topic:campaign-draft/*` | member `topics.json` declarations |
| `decision` | `decision:content-publish-proposal` | `team.json::operatingContract.decisionContexts` |
| `por` | `por:docs/marketing/STRATEGY.md` | filesystem and PoR document declarations |
| `external` | `external:operator` | `external_producers` or explicit external boundary |
| `team` | `team:monetization` | team registry |
| `process` | `process:learning-synthesis` | readability grouping only |
| `future` | `future:publish-telemetry` | target-state grouping only |

Node IDs are local Mermaid identifiers. Validators read node labels, not IDs.

## Qualifiers

Operating graph labels use the same qualifier semantics as marked inline references:

| Syntax | Meaning |
|---|---|
| `topic:foo/*` | Current live topic. Must resolve. |
| `topic[future]:foo/*` | Target-state topic. Absence is allowed. |
| `topic[old]:foo/*` | Historical topic. Not treated as live. |
| `topic[external]:foo/*` | Outside this team or repo. |

Do not use `topic[example]:` in contract graphs.

## Edge Semantics

In the first implementation, edge meaning is inferred from typed node kinds and direction:

| Edge | Required runtime match |
|---|---|
| `external -> topic` | A member drains that topic and declares the external producer. |
| `topic -> member` | The member declares matching `intake[]`, `required_read[]`, or `evidence_consumed[]`. |
| `member -> topic` | The member declares matching `output[]`. |
| `member -> decision` | The member declares `decisions_owned[]`, or may raise `capability-gap`. |
| `decision -> member` | The member declares `decisions_consumed[]` or evidence for that decision. |
| `member -> por` | The member declares a `por_file` output to that path. |
| `topic -> team` | A member output declares the topic with `destination_team`. |
| `process` / `future` edges | Allowed for readability; they do not satisfy runtime completeness. |

The graph intentionally treats `topic -> member` as a broad read edge. The exact read subtype remains in `topics.json`:

| Runtime field | Meaning |
|---|---|
| `intake[]` | Member drains actionable items from the topic. |
| `required_read[]` | Member receives the topic as always-on heartbeat context. |
| `evidence_consumed[]` | Member cites the topic when contributing to named decisions. |

This keeps Mermaid readable while preserving precise runtime semantics in structured config.

## Diff Relationships

Diff normalizes Mermaid edges and runtime config into semantic relationships before comparing them:

| Relationship | Mermaid shape | Runtime source |
|---|---|---|
| `topic_read` | `topic -> member` | `intake[]`, `required_read[]`, or `evidence_consumed[]` |
| `topic_output` | `member -> topic` | `output[]` |
| `por_output` | `member -> por` | `output[]` with `destination_kind: por_file` |
| `decision_owned` | `member -> decision` | `decisions_owned[]` |
| `decision_consumed` | `decision -> member` | `decisions_consumed[]` or `evidence_consumed[].for_decisions[]` |
| `capability_gap_raised` | `member -> decision:capability-gap` | `raises_capability_gaps` |
| `external_producer` | `external -> member` | `external_producers[]` |
| `external_producer_intake` | `external -> topic` | `external_producers[]` plus matching `intake[]` |
| `cross_team_output` | `topic -> team` | `output[].destination_team` |

Topic matching uses the same prefix overlap semantics as topic validation, so `campaign-draft/*` matches a more specific compatible prefix.

Example graph-to-runtime diff:

```text
[graph_relationship_missing_in_runtime] topic_read
docs/marketing/OPERATING_MODEL.md:355 says topic:marketing/notebook/* -> member:researcher.
Runtime has no matching researcher intake, required_read, or evidence_consumed declaration.
Suggested fixes:
- add required_read "marketing/notebook/*" to researcher/topics.json
- or remove the topic -> member edge from the operating graph
```

Example runtime-to-graph diff:

```text
[runtime_relationship_missing_in_graph] topic_output
scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/topics.json declares member:researcher -> topic:hook-record/*.
The contract graph does not show a matching relationship.
Suggested fixes:
- add member:researcher -> topic:hook-record/* to the operating graph
- or remove the runtime output declaration if it is obsolete
```

## Validation Rules

Initial rules:

| Rule | Severity | Meaning |
|---|---|---|
| `graph_unknown_member` | error | `member:` node not in the team contract. |
| `graph_unknown_decision` | error | `decision:` node not in any loaded decision context. |
| `graph_unknown_team` | warning | `team:` node not found in the team registry. |
| `graph_unknown_por` | error | `por:` path does not exist. |
| `graph_untyped_node` | error | Contract node lacks a typed machine label. |
| `graph_topic_unresolved` | error | Live `topic:` node has no matching runtime topic relationship. |
| `graph_future_topic_live_edge` | warning | Future topic appears on an active direct edge. |
| `graph_edge_unbacked` | error | Direct edge implies a runtime relationship absent from config. |
| `graph_declared_intake_missing` | error | Live intake in `topics.json` is missing from a contract graph. |
| `graph_declared_output_missing` | warning | Live output in `topics.json` is missing from a contract graph. |

Generated heartbeat prompt checks are paired with this layer: every active member should receive a generated `# Topic Contract` section from `topics.json`.

## Adoption Checklist

1. Add a full topic-level contract diagram to the team plan-of-record.
2. Add `prompt-manager-graph` metadata with `mode: contract`.
3. Type every contract node.
4. Keep conceptual joins as `process:` nodes.
5. Mark target-state-only topics with `topic[future]:`.
6. Run `prompt-manager graph operating-model validate --team <team>`.
7. Run `prompt-manager graph operating-model diff --team <team>`.
8. Reconcile findings by updating docs or runtime config after design review.

## Non-Goals

- Do not make Mermaid the runtime source of truth.
- Do not support arbitrary Mermaid syntax.
- Do not use operating graphs to bypass `topics.json`.
- Do not hide untyped contract nodes behind display labels.
- Do not auto-apply sync changes until validation has proven stable.
