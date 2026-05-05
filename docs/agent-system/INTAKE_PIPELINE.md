# Intake Pipeline

The canonical pattern for how external signals become structured evidence, decisions, skills, or capability-gap proposals. Cites `LAYERS.md`, `TEAM_MEMBER_ARCHITECTURE.md`, `PROMOTION_LADDER.md`, and `DECISIONS.md` (for what happens after the router files a decision). Validated by the [three pillars](PRIMITIVES.md#three-pillars-of-topic-validation): [`TOPICS_SCHEMA.md`](TOPICS_SCHEMA.md) (declarations), [`PROSE_SCAN_TARGETS.md`](PROSE_SCAN_TARGETS.md) (prose drift), [`RUNTIME_ATTRIBUTION.md`](RUNTIME_ATTRIBUTION.md) (observed writes).

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
      "taxonomy": "marketing-research",
      "classifier_skill": "marketing-signal-classifier",
      "source_team": null
    }
  ],
  "external_producers": ["vision-walk", "operator", "bookmark-intelligence-hub"]
}
```

`prefix` names the topic-prefix the member drains; `taxonomy` names the JSON sidecar (`docs/<domain>/<id>.json`) that owns the signal vocabulary, dispatch table, evidence rules, and destination schemas; `classifier_skill` is the optional pure-judgment skill loaded when assignment of `signal_type` requires interpretation; `source_team` is set when the prefix is fed by another team's members. The heartbeat builder generates the universal drain procedure into the member's prompt as the `# Inbox Flow` section.

---

## The inbox-router-drain pattern

The dominant mechanism for unstructured-to-structured signal flow. Used by `marketing-signal-classifier`, `monetization-signal-classifier`, `market-validation-triage`, and others (paired with their taxonomies).

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

### Two routing modes

The same drain procedure runs in either of two modes, picked per intake by whether the topic-prefix carries enough information to determine `signal_type` deterministically.

**Mode 1 — Classifier-required (judgment intake).** The producer writes a raw note under a generic prefix (`research-inbox/<signal-type>/<slug>`); the assigned signal-type is a *hint*, not authority. The drainer applies a portable classifier skill — pure judgment, no team or destination coupling — to derive the authoritative `signal_type`, `evidence_strength`, and `honesty_flags` for each entry. Use this mode when the producer is upstream of the taxonomy (operator, vision-walk, cross-team handoff) and may misclassify, or when interpretation is required to disambiguate signal types. Examples: `marketing-crew/researcher` → `marketing-signal-classifier` over `marketing-research`; `monetization/opportunity-scout` → `monetization-signal-classifier` over `monetization-opportunity`; `monetization/market-validator` → `market-validation-triage` over `monetization-validation`.

**Mode 2 — Deterministic prefix (no classifier).** The producer is constrained to write only valid taxonomy signal-types in the prefix segment after the inbox name (`<inbox-name>/<signal-type>/<slug>`), and that segment is taken as authoritative. No classifier skill is loaded; the drainer's heartbeat omits the classifier line and reads the signal-type straight from the topic. Use this mode when (a) the producer set is closed and trusted (typically same-team curation, not external alpha) and (b) the taxonomy's signal-types are mutually exclusive enough that misclassification is rare. Examples: `marketing-crew/brand-manager` and `meta-optimization/debt-curator` both drain `<team>/notebook/*` against the `notebook-debt` taxonomy with no classifier — the producer writes `notebook/promotion-candidate/<slug>` etc., and the curator trusts the prefix.

**Universal-source intakes pair with deterministic-prefix routing.** When `intake[].source_team = "*"` (any team's members may write — see `TOPICS_SCHEMA.md` § Universal-source intakes), the producer set is structurally open. Trust must be established at the producer side: a single writer skill (declared in `external_producers`) constrains the topic shape and signal-type assignment, and the drainer treats the prefix as authoritative. Adding a classifier skill on top of a universal-source intake is architectural drift — the writer skill already enforces shape, and the drainer must read the entry to start work anyway. Examples: `scenario-qa/bug-investigator` drains `bug-inbox/*` (universal-source) against `bug-report` taxonomy via `report-bug` writer skill; investigation includes signal-type validation as its first sub-step. The sister flow `meta-optimization/friction-curator` drains `friction-inbox/*` against `friction-report` taxonomy via `report-friction` writer skill; the curator validates scope (or reclassifies `unknown`) and routes to the appropriate scoped friction topic owned by an existing sub-member. Both flows establish the **universal observation flow** primitive — universal-source intake + writer skill + drainer + heartbeat trigger paragraph — as a reusable architectural pattern.

The choice is structural, recorded in `topics.json`: setting `intake[].classifier_skill` selects mode 1; omitting it selects mode 2. The heartbeat builder renders the appropriate procedure either way; the drainer does not need to know which mode is in use beyond what the generated section says.

### Routing — drain duty

The draining member uses its taxonomy (`intake[].taxonomy`) and, when judgment is needed, its classifier skill (`intake[].classifier_skill`) to **route each entry exactly once**. The heartbeat builder generates a universal `# Inbox Flow` section that names the prefix, the loaded taxonomy, the classifier (if any), the destination schemas, and the dispatch table. The classifier is portable and team-agnostic — it returns a recommendation; the member's drain procedure picks the action.

After routing, the entry must leave the inbox view (must no longer carry an `<inbox-name>/*` prefix) in one of two ways:

- **Promoted to permanent canon** — retag the entry to its destination prefix:

  ```bash
  prompt-manager team knowledge-update <team> <id> --topic="<destination-topic>"
  ```

  e.g., `research-inbox/audience/foo` → `audience-scan/foo`. Destination topics use the canonical prefix for the surface (`audience-scan/<slug>`, `competitor-record/<slug>`, `hook-record/<slug>`, `monetization-benchmark-adjacent-record/<slug>`, etc.).

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

Method skills are loaded by the draining member on the strength of the classifier's signal-type recommendation (or the deterministic topic-prefix). The classifier itself is not a method skill; it returns judgment only — the member dispatches.

---

## Promotion / Routing — what happens to the output

The draining member (using its taxonomy's `actionSelection` set, possibly informed by a classifier recommendation) chooses the smallest useful action:

| Condition | Action |
|---|---|
| Weak one-off signal | Drop/delete the inbox entry after noting it in handoff if useful. If the weak signal has real audit value, retag it to a non-inbox audit prefix such as `low-signal/<slug>` or a domain-specific equivalent. Never leave routed material under `<inbox-name>/*`. |
| Concrete sourced observation | Add a knowledge entry under the canonical surface prefix (e.g., `audience-scan/<slug>`, `competitor-record/<slug>`). |
| Capability already exists for this signal | Run the existing skill or Action; route the output as a knowledge observation. |
| Trivial automation, no LLM judgment needed | Create + run a new Action (no decision required — see `DECISIONS.md` §4). |
| Repeated but unresolved pattern | File a `meta-self-improvement` decision proposing a skill / scenario / config change. The "notebook debt" surface is, in the live architecture, this kind of decision — not a markdown file. |
| Converging evidence meets threshold | Raise the owned decision context (e.g., `audience-update`, `channel-strategy-update`). |
| Repeatable method has no skill | Propose a skill through the meta-optimization path (`skill-optimizer`). |
| Collection requires missing source/tool/scenario | Raise `capability-gap` (see `DECISIONS.md` §5 for the capability-gap-vs-decision criteria). |
| Signal belongs to another domain | Write to that domain's prefix or hand off as cross-team flow (cross-team output ownership rules in `DECISIONS.md` §9). |

The drain's execute-directly vs file-decision threshold (and what governs each routing-outcome row above) is defined in `DECISIONS.md` §4. This table names what each row *is*; that file names *when* the member is allowed to take each row vs escalate.

Each member declares structurally what it produces:

```jsonc
{
  "output": [
    { "prefix": "audience-scan/*",                  "destination_kind": "knowledge", "destination_team": null,           "schema": "audience-scan" },
    { "prefix": "monetization-benchmark-adjacent-record/*", "destination_kind": "knowledge", "destination_team": "monetization", "schema": "monetization-benchmark-adjacent" }
  ],
  "decisions_owned": ["audience-update", "channel-strategy-update"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true
}
```

`destination_kind` is one of `knowledge`, `decision`, `por_file`, `capability_gap`, `skill_proposal`, `backlog`. `schema` references a front-matter shape declared on the producer's taxonomy (`taxonomy.schemas.<id>`). Cross-team flow is declared on both sides — see [Cross-team schema ownership](#cross-team-schema-ownership) below and `TOPICS_SCHEMA.md`.

### Cross-team schema ownership

When a prefix crosses team boundaries, **the producer's taxonomy owns the front-matter schema**. The consumer adopts the same schema; it does not redefine it. This is one of the load-bearing rules of the inbox-flow architecture and the one most likely to confuse a new adopter.

| | Producer side | Consumer side |
|---|---|---|
| `topics.json` field | `output[].destination_team = "<other-team>"`, `output[].schema = "<id>"` | `intake[].source_team = "<producer-team>"`, `intake[].taxonomy = "<consumer-domain>"` |
| Owns front-matter shape | yes | no |
| Owns dispatch / routing on read | no | yes |
| Validator behavior | `missing_destination_schema` resolves `output[].schema` against the producer's taxonomy | `unknown_taxonomy` resolves the consumer's intake taxonomy independently |

Worked example. `marketing-crew/researcher` writes `monetization-benchmark-adjacent-record/*`. Its `topics.json` carries `output: [{ "prefix": "monetization-benchmark-adjacent-record/*", "destination_team": "monetization", "schema": "monetization-benchmark-adjacent" }]`. The schema id resolves under the *marketing-research* taxonomy (`docs/marketing/signal-taxonomy.json#schemas.monetization-benchmark-adjacent`). On the receiving side, `monetization/market-validator` declares `intake: [{ "prefix": "monetization-benchmark-adjacent-record/*", "taxonomy": "monetization-validation", "source_team": "marketing-crew" }]`. The consumer's `monetization-validation` taxonomy governs how `market-validator` classifies and routes the entry on read; it does not control the on-disk shape — the producer already set that.

Why this rule: the on-disk shape is fixed at write time. The producer is the only party who can guarantee shape consistency; if the consumer redefined the schema, the producer would be unable to validate its own writes. Routing is a read-side concern and may legitimately differ across consumers (a single prefix could be drained by multiple consumers under different taxonomies later). Schemas can't.

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

`topics.json` is the structural declaration of how each member participates in the pipeline. Each member declares topic-prefix relationships across four kinds:

- `intake[]` — prefixes the member drains. The classifier/triage skill named here owns routing for those entries.
- `required_read[]` — prefixes the member must read every heartbeat (rendered into the active-task brief's "## Required Memory" section). Reading without draining.
- `evidence_consumed[]` — prefixes the member cites as evidence when authoring decisions in a named `for_decisions[]` context. Reading with explicit decision provenance.
- `output[]` — prefixes the member writes (with `destination_kind`, optional `destination_team`, schema, and supersession policy).

The graph loader builds a directed graph from these declarations across all teams; the validator (`prompt-manager graph topics`) cross-checks the graph for orphan_input, orphan_output, conflicting_drain, unread_required, dangling_evidence_decision, dangling_por_sink, missing_destination_schema, wildcard_source_misuse, topic_key_prefix_mismatch, stalled_drain, and piling_inbox. The full rule list and severities live in `TOPICS_SCHEMA.md` § Validation rules.

The layer-audit skill consumes `topics.json` programmatically to derive Intake / Collection / Promotion / Routing scores instead of prose grep — closing the loop where the layered architecture audits itself structurally rather than narratively.

---

## When the pipeline does not apply

Not every member is signal-shaped. A pure reviewer, a code-writing scenario engineer, or a deterministic-CLI maintainer may have no intake topics, no collection layer, and no promotion mechanism. For these members:

- `topics.json` is `{}` (or omitted), explicitly declaring no flow.
- The audit skill scores Intake / Collection / Analysis / Promotion as `n/a` rather than `missing`.

Use the layer model honestly. Members that *should* have a pipeline but don't are smells; members that *don't need* one are not.
