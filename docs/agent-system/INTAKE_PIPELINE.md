# Intake Pipeline

The canonical pattern for how external signals become structured evidence, decisions, skills, or capability-gap proposals. Cites `LAYERS.md`, `TEAM_MEMBER_ARCHITECTURE.md`, `PROMOTION_LADDER.md`, and `DECISIONS.md` (for what happens after the router files a decision).

This is the live model. The historical "hot buffer / living notebook / permanent structure" three-tier framing is retired in favor of this one.

---

## The pipeline

```text
Intake  ->  Collection  ->  Analysis Method  ->  Promotion / Routing
```

Four stages. Each stage has a home (per `LAYERS.md`); each is independently auditable (per `TEAM_MEMBER_ARCHITECTURE.md`). Stages are not always implemented as separate skills — combined skills are allowed early — but the conceptual stages exist whether or not the implementation acknowledges them.

---

## Intake — how work enters

Work enters a member's lane through one of:

- **Operator-fed.** Vision walk, direct instruction, alpha intake.
- **Proactive.** Scheduled scan, known source list, telemetry.
- **Cross-team.** Decisions, inbox messages, handoff, capability-gap routed from elsewhere.
- **Internal.** Logs, knowledge entries, scenario metrics.

Each intake channel a member drains is declared structurally in `topics.json`:

```jsonc
{
  "intake": [
    {
      "prefix": "research-inbox/*",
      "drained_by_skill": "marketing-research-router",
      "source_team": null
    }
  ],
  "external_producers": ["vision-walk", "operator", "bookmark-intelligence-hub"]
}
```

`prefix` names the topic-prefix the member drains; `drained_by_skill` names the router skill that owns the procedure; `source_team` is set when the prefix is fed by another team's members.

---

## The inbox-router-drain pattern

The dominant mechanism for unstructured-to-structured signal flow. Used by `marketing-research-router`, `monetization-opportunity-router`, `market-validation-router`, and others.

### Inbox topics

Inbox entries live as **team knowledge entries** under a hierarchical topic prefix:

```
<inbox-name>/<signal-type>/<short-slug>
```

Examples: `research-inbox/audience/foo`, `opportunity-inbox/competitor-move/bar`. There is no separate JSONL file or notebook markdown — the inbox is a query against team knowledge:

```bash
prompt-manager team knowledge-list <team> --topic-prefix=<inbox-name>/
prompt-manager team knowledge-list <team> --topic-prefix=<inbox-name>/<signal-type>/
```

### Adding to the inbox

Producers (operator, vision-walk, cross-team handoff) add entries via:

```bash
prompt-manager team knowledge-add <team> \
  --by=<source-id> \
  --topic="<inbox-name>/<signal-type>/<short-slug>" \
  --content="<raw note + honesty/confidence flags + optional suggested-method>" \
  --source="<original-url-if-known>"
```

`--by` identifies the source (e.g., `vision-walk`, `researcher`, `operator`); `--source` carries the original URL when available.

Each entry must preserve: source URL when available, raw operator note, confidence/honesty flags, and a proposed next method (or a reason no follow-up is warranted). Signal-type lives in the topic prefix, not the content.

### Routing — drain duty

The router skill drains the inbox by **routing each entry exactly once**. After routing, the entry must leave the inbox view (must no longer carry an `<inbox-name>/*` prefix) in one of two ways:

- **Promoted to permanent canon** — retag the entry to its destination prefix:

  ```bash
  prompt-manager team knowledge-update <team> <id> --topic="<destination-topic>"
  ```

  e.g., `research-inbox/audience/foo` → `audience-scan/foo`. Destination topics use the canonical prefix for the surface (`audience-scan/<slug>`, `competitor/<slug>`, `hook/<slug>`, `monetization-benchmark-adjacent/<slug>`, etc.).

  If the routed action creates a *new* entry on a different surface (decision, capability-gap), delete the inbox row instead of retagging.

- **Dropped** as weak / duplicate / out-of-scope:

  ```bash
  prompt-manager team knowledge-delete <team> <id>
  ```

The inbox view (`--topic-prefix=<inbox-name>/`) is therefore always the **unrouted set**. Permanent canon never uses any topic under `<inbox-name>/`; it lives under whichever destination prefix matches the routed surface.

### Drain conflicts

Two members must not both claim drain duty for overlapping prefixes — that's the `conflicting_drain` validation rule. Either:

- one member drains the broader prefix and the other drains a narrower (non-overlapping) sub-prefix, or
- one member drains and the other consumes routed output.

The validation rules in `prompt-manager graph topics` flag overlap as a structural error.

---

## Collection — how source material is gathered

Collection turns an intake claim into actual evidence. Modes:

- **Supplied source references.** Operator-provided links, bookmark exports, source URLs in the knowledge entry.
- **Manual web research with cited sources.** Default fallback; label findings honestly.
- **Scenario API or CLI.** When a Vrooli-controlled tool can fetch the source (e.g., `bookmark-intelligence-hub` for archived bookmarks), prefer that.
- **Action.** When the collection is a single deterministic command, expose it as an Action (per `PROMOTION_LADDER.md`).

When source access requires credentials, scheduling, or scraping that no controlled tool covers, file a `capability-gap` decision rather than fudging the collection. Do not pretend the scan happened.

### Collection discipline

- Prefer supplied source references and tool/Action output over manual web research.
- Do not create platform-specific intake skills when a general collection layer (e.g., bookmark-intelligence-hub) would cover the source.
- Label single-snapshot findings `light-interpretation` so downstream consumers know the evidence weight.
- If source access is blocked, raise a capability gap instead of pretending the scan happened.

---

## Analysis Method — how material is interpreted

A reusable method skill turns gathered evidence into a structured observation. Examples (from the marketing domain):

- `alpha-extraction`
- `audience-pain-mining`
- `workflow-deconstruction`
- `competitor-positioning-scan`
- `channel-format-scan`
- `hook-pattern-mining`
- `offer-and-funnel-scan`
- `skill-opportunity-scan`
- `benchmark-adjacent-scan`

Combined skills are allowed early — a method skill may both collect and analyze when the source is simple. Split collection into its own tool/skill/Action when source access becomes reusable, credentialed, scheduled, or deterministic.

Method skills are loaded by the router on the strength of the signal type. The router itself is not a method skill; it classifies and dispatches.

---

## Promotion / Routing — what happens to the output

The router (or the method skill it delegates to) chooses the smallest useful action:

| Condition | Action |
|---|---|
| Weak one-off signal | Drop/delete the inbox entry after noting it in handoff if useful. If the weak signal has real audit value, retag it to a non-inbox audit prefix such as `low-signal/<slug>` or a domain-specific equivalent. Never leave routed material under `<inbox-name>/*`. |
| Concrete sourced observation | Add a knowledge entry under the canonical surface prefix (e.g., `audience-scan/<slug>`, `competitor/<slug>`). |
| Capability already exists for this signal | Run the existing skill or Action; route the output as a knowledge observation. |
| Trivial automation, no LLM judgment needed | Create + run a new Action (no decision required — see `DECISIONS.md` §4). |
| Repeated but unresolved pattern | File a `meta-self-improvement` decision proposing a skill / scenario / config change. The "notebook debt" surface is, in the live architecture, this kind of decision — not a markdown file. |
| Converging evidence meets threshold | Raise the owned decision context (e.g., `audience-update`, `channel-strategy-update`). |
| Repeatable method has no skill | Propose a skill through the meta-optimization path (`skill-optimizer`). |
| Collection requires missing source/tool/scenario | Raise `capability-gap` (see `DECISIONS.md` §5 for the capability-gap-vs-decision criteria). |
| Signal belongs to another domain | Write to that domain's prefix or hand off as cross-team flow (cross-team output ownership rules in `DECISIONS.md` §9). |

The router's execute-directly vs file-decision threshold (and what governs each routing-outcome row above) is defined in `DECISIONS.md` §4. This table names what each row *is*; that file names *when* the router is allowed to take each row vs escalate.

Each member declares structurally what it produces:

```jsonc
{
  "output": [
    { "prefix": "audience-scan/*", "destination_kind": "knowledge", "destination_team": null },
    { "prefix": "monetization-benchmark-adjacent/*", "destination_kind": "knowledge", "destination_team": "monetization" }
  ],
  "decisions_owned": ["audience-update", "channel-strategy-update"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true
}
```

`destination_kind` is one of `knowledge`, `decision`, `por_file`, `capability_gap`, `skill_proposal`, `backlog`. Cross-team flow is declared on both sides — see `drafts/topics-schema.md`.

---

## Evidence rules

These apply at every stage of the pipeline:

- **Never fabricate** engagement, revenue, conversion, audience-size, or pricing facts.
- **One source is an observation, not canon.**
- **Three converging scans** can justify a decision when the sources are meaningfully independent.
- **Single-snapshot data must be labeled** `light-interpretation`.
- **Tool classifications are inputs, not proof.** Bookmark-intelligence-hub or other automated tagging may surface candidates; the analyst still evaluates relevance and evidence quality.
- **Researchers do not edit canon directly.** They propose changes for operator review via the appropriate decision context.

---

## How `topics.json` connects

`topics.json` is the structural declaration of how each member participates in the pipeline. Its loader (Phase 2 of the agent-system migration) builds a directed graph from these declarations; the validation rules (Phase 3) detect orphan inputs, orphan outputs, drain conflicts, dangling sinks, stalled drains, and piling inboxes.

The audit skill (Phase 5) consumes `topics.json` programmatically to derive Intake / Collection / Promotion / Routing scores instead of prose grep — closing the loop where the layered architecture audits itself structurally rather than narratively.

---

## When the pipeline does not apply

Not every member is signal-shaped. A pure reviewer, a code-writing scenario engineer, or a deterministic-CLI maintainer may have no intake topics, no collection layer, and no promotion mechanism. For these members:

- `topics.json` is `{}` (or omitted), explicitly declaring no flow.
- The audit skill scores Intake / Collection / Analysis / Promotion as `n/a` rather than `missing`.

Use the layer model honestly. Members that *should* have a pipeline but don't are smells; members that *don't need* one are not.
