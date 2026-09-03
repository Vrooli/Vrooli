# Topics

**Status:** canon. The human registry of what topic prefixes exist in the system, who produces and drains each, and what they're for. Sibling of `TOPICS_SCHEMA.md` (which defines the on-disk shape of `topics.json`); this file is the contents-of-the-cylinder view.

The mermaid in `README.md` shows a `Team inbox topics` cylinder feeding a `Router skill` diamond. This file lists what's inside that cylinder. The skill that drains it is the per-taxonomy classifier or triage skill — see § How topics are processed.

---

## What a topic is

A **topic** is a hierarchical string prefix used as the addressing scheme for team-knowledge entries. The prefix `audience-scan/*` declares a bucket; an entry under it (e.g., `audience-scan/foo`) is one message addressed by its unique key.

The set of declared topics across the system is finite and tractable. Every topic in active use has a structural declaration in some member's `topics.json` (someone writes it, someone drains it, or both — usually both). Topics that aren't declared anywhere are either drift (left over from deleted members) or unbuilt (someone *should* be draining them but isn't).

Topics are **scoped to a team**. Each team has its own knowledge store; when a producer on team A writes `audience-scan/foo`, that entry lives in team A's knowledge store. Cross-team flow — team B reading a topic owned by team A — is declared explicitly on both sides (`output[].destination_team` on the producer; `intake[].source_team` on the consumer). See `TOPICS_SCHEMA.md` § Cross-team flow for the rule, and `INTAKE_PIPELINE.md` § Cross-team schema ownership for the producer-owns-schema invariant.

## What a topic is not

- **Not a database table.** Entries are key/value records in team knowledge, not relational rows.
- **Not a folder.** Topics don't nest like filesystem paths despite the slashes; they're flat strings interpreted as prefixes.
- **Not a queue.** No head/tail semantics; the drainer routes per entry in any order.
- **Not the same as the `path:api/topics/` package's "topics."** That package serves a different concept (parent/child content-taxonomy trees with attached skills) and predates the inbox-flow data layer. Naming collision; different concern. See `README.md` § Naming note.

## How topics are processed

Every topic that has a drainer — any topic declared in some member's `intake[]` — is processed by that member through a **uniform action set**. The drainer reads each entry, applies its taxonomy (and optionally a classifier skill — see `INTAKE_PIPELINE.md` § Two routing modes), and chooses exactly one outcome from the set enumerated in `INTAKE_PIPELINE.md` § Promotion / Routing (drop, observe/retag-to-canon, promote-to-canon, route-to-another-topic, file-work, file-backlog/file-initiative).

**The action set is uniform across the system.** Whatever produces an entry — operator alpha, vision-walk brainstorm, scheduled scan, cross-team handoff, proactive self-generation, another agent's `route-to-another-topic` outcome — the receiving drainer applies this same set. When the action is `route-to-another-topic`, the receiving drainer applies the set again to the new entry. The architecture is recursive: every entry is processed by *some* drainer through the same outcome set until it terminates in drop / observe / canon / Swarm Manager work.

This is the load-bearing claim that makes the substrate auditable. Adding a new input shape doesn't require new architecture — only a topic + a drainer + a taxonomy entry. See `INTAKE_PIPELINE.md` § Promotion / Routing for the full table of when each outcome is allowed, and `SWARM_MANAGER_WORK.md` §4 for the direct-edit-vs-swarm-manager threshold.

The drainer's classifier/triage skill (when one exists) is loaded by the heartbeat builder from the member's `intake[].classifier_skill` and lives at `path:scenarios/prompt-manager/store/skills/packs/core/<id>/`. Today's classifier is `signal-classifier`, parameterized over each member's `intake[].taxonomy`. Deterministic-prefix intakes need no classifier when the writer skill or producer-side contract already enforces the signal type.

For **universal-source intakes** (`intake[].source_team = "*"` — any team's members may write; today: `bug-inbox/*` on scenario-qa, `friction-inbox/*` on meta-optimization), a producer-facing trigger paragraph is rendered into every member's heartbeat prompt so producers know when to invoke the writer skill. The declaration semantics, validator behavior, trigger-rendering mechanics, and the data-driven-rendering proposal for a third such intake all live in `TOPICS_SCHEMA.md` § Universal-source intakes.

## When to create a topic

Create a new topic when **all three** of these are true:

1. A member needs to declare structurally that it produces or consumes a kind of work.
2. That kind of work doesn't already match an existing topic prefix. When in doubt, run `prompt-manager team knowledge-list <team> --topic-prefix=<candidate>` and skim what's there.
3. The work is **signal-shaped** — discrete entries that go through a lifecycle, not ad-hoc text or per-task chatter.

Do **not** create a topic for:

- One-off communication between agents (use direct member-to-member channels or the team-inbox messaging system, which is a different surface — see `PRIMITIVES.md`).
- Configuration or static documentation (PoR or member files).
- Pure logging without intent to drain (scenario logs or telemetry).
- Parallel naming for an existing topic (e.g., creating `topic[example]:competitor-scan/*` when `topic[example]:competitor-record/*` already exists). Naming inconsistencies that survive go in § Naming conventions to be reconciled, not minted as new topics.

When you do create one: declare it in the producer's `topics.json#output[]` and on the consuming side. The consumer-side declaration is whichever fits:

- `intake[]` if the consumer drains and routes entries through the inbox-router-drain pattern.
- `required_read[]` if the consumer must read entries every heartbeat without draining (rendered into "## Required Memory").
- `evidence_consumed[]` if the consumer cites entries as evidence on a named work type.

If no consumer is identified yet, that's a smell: file a `capability-work` work item in Swarm Manager or onboard a consumer. Don't let an undrained topic accumulate orphaned entries; the validator will flag it as `orphan_output`. Conversely, if a consumer's `required_read[]` references a prefix nobody outputs, the validator surfaces it as `unread_required`. Both rules read from the same declaration substrate; see `TOPICS_SCHEMA.md` § Validation rules for the full set.

---

## Naming conventions

Topics use kebab-case, are scoped to a team's knowledge store unless cross-team flow is declared, and pick a prefix shape based on what the entries are *for*. The conventions below describe what's in active use; some are stable, some surface known inconsistencies that meta-optimization should resolve via decision.

### Stable conventions

- **kebab-case.** Lowercase ASCII letters, digits, and hyphens. No underscores, no camelCase.
- **Wildcard.** `<prefix>/*` matches any entry whose key starts with `<prefix>/`. Bare `*` is disallowed by the validator.
- **Slug last.** Every full entry key ends in a unique slug under the prefix (e.g., `audience-scan/<slug>`). Slugs are short, kebab-case, descriptive.
- **One topic, one purpose.** A topic prefix is for one kind of work. Don't co-mingle audience observations and competitor pricing under a single prefix because they're in the same domain — they're different signal types and need different routing.
- **Slash-slug entry keys.** Every knowledge entry's `topic` field is `<declared-prefix>/<slug>` — the prefix matches some member's `topics.json` declaration with any trailing `/*` stripped, the slug names the entry under that prefix. Date-rotated snapshots use the date as the slug (`audience-scan/2026-04-23`); per-case reports use a descriptive slug (`bug-investigation-report/login-500-2026-05-03`). Flat-hyphenated keys (`audience-scan-2026-04-23`) and bare tokens with no slash structure are forbidden — they don't match prefix-based queries (`prompt-manager team knowledge-list <team> --topic-prefix=<prefix>/` returns nothing for a flat key). The `topic_key_prefix_mismatch` validation rule (severity warning) flags any entry whose topic doesn't match a declared prefix on its team.
- **Closed-vocabulary type suffix on canonical surfaces.** Every canonical-surface (durable) topic prefix has the shape `<subject>-<type>/*`, where `<type>` is drawn from a fixed eight-word vocabulary and `<subject>` is any kebab-case noun phrase that is not itself in the vocabulary. The vocabulary:
    | Suffix | Entry shape | Schema family |
    |---|---|---|
    | `-scan` | Sampled / survey-shaped observation, single-snapshot or batch | scan-shaped front-matter |
    | `-audit` | Exhaustive / adversarial review with findings against a standard | audit-shaped |
    | `-record` | Durable entity record (one entity per entry) | record-shaped |
    | `-draft` | In-flight artifact that becomes a record or canon edit | draft-shaped |
    | `-log` | Append-only event entries | log-shaped |
    | `-canon` | Proposed / applied PoR edit (`destination_kind = por_file`) | canon-edit-shaped |
    | `-report` | Investigative finding with structured rationale | report-shaped |
    | `-prep` | Synthesis briefing for a downstream consumer | prep-shaped |
    Mechanical constraint: the `<subject>` must NOT itself be a vocabulary word, so `topic[example]:quality-audit/*` is fine (`quality` is the subject, `audit` is the suffix), but `topic[example]:audit-audit/*` is rejected. Picking the right suffix is structural, not aesthetic — `*-audit/*` carries audit-shaped front-matter, `*-scan/*` carries scan-shaped, and `*-finding` / `*-violation` / `*-gap` carry adversarial evidence. Reports vs. records: a report carries reasoning (a finding + evidence + rationale); a record carries one entity's state (one row, no narrative). Logs are append-only event streams; preps are synthesis briefings explicitly authored for a downstream consumer.
- **Inbox / synthesis topics use purpose suffixes, not type suffixes.** The vocabulary above is for *canonical* surfaces. Transient and synthesis surfaces use `*-inbox/*` when a drainer must route entries per item.
- **Team name in the prefix only when the team name is part of the *concept*.** Team scoping is implicit — every entry already lives in some team's knowledge store, and cross-team flow is declared explicitly via `destination_team` / `source_team`. Adding a team name to the prefix is therefore redundant by default, and *coupling* the topic to a particular team makes it harder to reorganize. The default is **no team in the prefix** — `audience-scan/*`, `competitor-record/*`, `bug-inbox/*`, `qa-run/*`. Add the team only when the team name is genuinely part of the concept, such as `marketing-canon/*` and `monetization-canon/*`, which are distinct PoR-write surfaces that need to coexist as named concepts.

### Topic shapes (current usage)

| Shape | Pattern | Examples | Lifetime |
|---|---|---|---|
| **Inbox (transient)** | `<inbox-name>/<signal-type>/<slug>` | `research-inbox/audience/foo`, `opportunity-inbox/competitor-move/bar`, `bug-inbox/regression/baz`, `friction-inbox/toolchain/qux` | drained → entry retagged or deleted |
| **Scan (survey-shaped observation)** | `<subject>-scan/<slug>` | `audience-scan/foo`, `debt-scan/bar`, `topic[example]:quality-audit/baz` (audit, not scan — see below) | durable; the entry's permanent home |
| **Audit (adversarial / compliance review)** | `<subject>-audit/<slug>` | `topic[example]:quality-audit/foo`, `toolchain-audit/bar`, `runtime-health-audit/baz`, `platform-code-audit/qux`, `skill-audit/quux`, `action-audit/...`, `team-audit/...`, `agent-audit/...` | durable; not retagged |
| **Record (entity record)** | `<subject>-record/<slug>` | `competitor-record/foo`, `hook-record/bar`, `candidate-sku-record/baz`, `monetization-benchmark-record/qux`, `topic[example]:outcome-target-record/quux`, `initiative-portfolio-record/...`, `friction-triage-record/...` | durable; one entity per entry |
| **Draft (in-flight artifact)** | `<subject>-draft/<slug>` | `campaign-draft/foo` | durable until promotion to canon or record; mutable until then |
| **Ledger (sweep coverage memory)** | `<subject>-visited/<id>` | `skill-visited/<skill-id>`, `action-visited/<action-id>`, `team-visited/<team-id>`, `agent-visited/<agent-id>` | durable; one entry per target, superseded on re-visit |
| **Log (append-only events)** | `<subject>-log/<slug>` | `publish-history/foo`, `monetization-ledger-log/bar`, `qa-run/baz` (run is implicit log; treated as such) | durable, append-only |
| **Canon (PoR edit)** | `<team>-canon/<slug>` | `marketing-canon/foo` (→ docs/marketing/strategy/STRATEGY.md or AUDIENCES.md), `monetization-canon/bar` (→ docs/monetization/catalogs/CATALOG.md) | translates to a PoR markdown edit; `destination_kind = por_file` |
| **Prep (synthesis briefing)** | `<subject>-prep/<slug>` | `topic[example]:workshop-decision-prep/2026-05-03` | durable; consumed by a downstream synthesizer or operator |
| **Cross-team flow** | same as any of the above, with explicit `destination_team` / `source_team` | `monetization-benchmark-adjacent-record/foo` (marketing → monetization) | depends on the receiving member's drain |

### Residual Signals

There is no residual catch-all topic. If a signal does not fit an existing typed inbox, the producer chooses the closest structural route:

- code/scenario defect -> `bug-inbox/*`;
- system friction, workaround, repeated manual pain, or process leak -> `friction-inbox/*`;
- missing capability that blocks work -> `capability-work` work item;
- domain evidence -> the owning domain's typed inbox or canonical observation prefix;
- durable truth change -> owning work type.

If none of those routes fits, file `meta-self-improvement` to add the missing typed topic or taxonomy. Do not create an undrained holding area.

### Contrarian challenge topics

Contrarian topics are work-item-feedback surfaces, not a global operator inbox. The canonical lifecycle lives in [`REVIEW_FEEDBACK.md`](REVIEW_FEEDBACK.md).

- Authors do not drain these topics as `intake[]`. They consume them as work-item evidence via `evidence_consumed[]` for the contexts they own.

---

## Per-team topic registry

Per team: the topics that team currently produces and drains, with first-principles observations on gaps. "Current state" is derived directly from each member's `topics.json`. "Observations" are draft notes; resolution is workshop-pending unless flagged as a structural error.

### director-swarm

**Mission:** keep Vrooli's initiative portfolio flowing through swarm-manager and surface outcome-driven strategy. Director teams consume Swarm Manager work (not topics) and produce prep artifacts for downstream consumption.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `outcome-strategist` | _(none — proactive; reads Swarm Manager work)_ | `outcome-target-record/*` | — |
| `portfolio-manager` | _(none — proactive; reads Swarm Manager work)_ | `initiative-portfolio-record/*` | — |
| `vision-walk-prep` | _(none — proactive; reads Swarm Manager work)_ | `vision-walk-record/*`, plus produces into other teams' inboxes (see below) | writes `research-inbox/*` → marketing-crew, `opportunity-inbox/*` → monetization, `validation-inbox/*` → monetization |

**Observations (draft):**
- `vision-walk-prep` is the canonical example of a **synthesis pipeline** member: it consumes Swarm Manager work state from other teams and produces (a) its own `vision-walk-record/*` artifact for the operator, and (b) seeded entries directly into other teams' inboxes for those teams to drain after the walk. The seeded-inbox edges are declared structurally on each receiving member's `intake[].source_team = "director-swarm"`; a human-readable inputs registry that would catalog this producer as a first-class flow is workshop-pending (see `README.md` § Mental Model).
- No director-swarm member has `intake[]` — confirms the user's mental model that director teams pull decisions rather than drain topics. This is correct, not a gap.
- `outcome-target-record/*` and `initiative-portfolio-record/*` are consumed by the operator (and possibly by `swarm-manager`) but no `topics.json` declares the consumer. If a future member or scenario consumes them programmatically, declare it via `intake[].source_team = "director-swarm"`.

### infra-health

**Mission:** protect Vrooli's local platform; turn runtime signals + platform-code audits + contrarian review into operator-routed reliability work.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `runtime-health-scanner` | _(none — proactive)_ | `runtime-health-audit/*` | — |
| `platform-code-auditor` | _(none — proactive)_ | `platform-code-audit/*` | — |

**Observations (draft):**
- All three members are pure proactive producers; none have `intake[]`. This is consistent with infra-health's mission (audit-driven, not signal-driven).
- No `topic[future]:infra-inbox/*` exists. If the operator wants to push infra-relevant alpha into this team, no current topic catches it. **Possible gap:** a `topic[future]:infra-inbox/*` for operator-fed runtime/platform alpha, drained by `runtime-health-scanner` or a new triage member. Workshop.
- Contrarian output follows the shared challenge lifecycle in § Contrarian challenge topics.

### marketing-crew

**Mission:** own Vrooli's external voice — subscription marketing, OSS narrative, brand canon, publishing pipeline.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `producer` | _(none — draws open work from `content-desk` campaigns)_ | `audience-scan/*`, `competitor-record/*`, `hook-record/*`, `workflow-scan/*`, `skill-scan/*`, `channel-scan/*`, `format-scan/*`, `monetization-benchmark-adjacent-record/*` | writes `monetization-benchmark-adjacent-record/*` → monetization |
| `brand-manager` | `marketing-craft-observation/*` | `marketing-canon/*` (por_file → `path:docs/marketing/strategy/STRATEGY.md`, `path:docs/marketing/strategy/AUDIENCES.md`), `brand-snapshot/*` | — |
| `marketing-contrarian` | _(none — proactive; reads peer decisions)_ | _(none declared; challenge reports are the contrarian-review sidecar per `REVIEW_FEEDBACK.md`)_ | — |

**Observations (draft):**
- Drafts, claims, campaigns, coverage and publish history are no longer topics on this team. They are records in the `content-desk` scenario, and account state is in `channel-manager` (`team-capability-consolidation`, applied 2026-07-28). The former `campaign-draft/*` and `foo-publish-history/*` prefixes retired with the `oss-advertiser`, `subscription-advertiser` and `publisher` members; do not reintroduce them as topics.
- `producer` declares no intake. That is the correct end state for a member whose work queue lives in a scenario rather than in a topic — the campaign slot is the queue. Read it as an intentional empty, not a missing declaration.
- `marketing-canon/*` is unusual: two `output[]` entries on the same prefix with different `destination_path` values (one to STRATEGY.md, one to AUDIENCES.md). This is a known pattern for por_file topics — the prefix names the *category*, the destination_path names the file. Document the pattern explicitly in § Stable conventions if it's the intended design.

### meta-optimization

**Mission:** apply evolutionary pressure to Vrooli's meta-layer — skills, agents, teams, tool contracts.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `debt-curator` | _(none — reads friction, audits, and decisions)_ | `debt-scan/*` | — |
| `skill-optimizer` | _(none — proactive)_ | `skill-audit/*`, `action-audit/*` | — |
| `team-agent-optimizer` | _(none — proactive)_ | `team-audit/*`, `agent-audit/*` | — |
| `toolchain-validator` | _(none — proactive)_ | `toolchain-audit/*` | — |
| `run-introspector` | _(none — proactive)_ | `run-lesson-report/*` | — |
| `friction-curator` | `friction-inbox/*` (taxonomy: `friction-report`, source: `*` — universal-source) | `friction-report/<scope>/*` (delivered to scoped sub-member topics), `friction-triage-record/<YYYY-MM-DD>` | producers: every team via the `report-friction` writer skill |

**Observations (draft):**
- Five proactive auditors plus one contrarian plus one friction router. This is the densest team.
- `friction-inbox/*` is one of the system's two **universal-source intakes** (semantics in `TOPICS_SCHEMA.md` § Universal-source intakes; sister flow is `bug-inbox/*` on scenario-qa). The friction-curator validates scope, reclassifies `unknown`, and routes by writing to the existing `friction-report/<scope>/<date>/<slug>` topics on the scoped-topic owners' behalf. Curator owns no work types — routing is determinate; capability-works are still raised by the scoped-topic owners.
- `friction-triage-record/<YYYY-MM-DD>` is a daily snapshot, supersedesPrevious=true, drained by debt-curator (synthesis input) and by operator review. `orphan_output` warning is by-design here (no peer drainer).
- `run-lesson-report/*` is consumed by `meta-contrarian` as evidence for Swarm Manager work and by other optimizers' input gathering, but no `intake[]` declares it. The flow goes through the unified work stream, not a direct topic-drain.
- The four scoped friction topics (`friction-report/toolchain/*`, `friction-report/run-execution/*`, `friction-report/prompt-team-agent-storage/*`, `friction-report/recurring-workaround/*`) are now multi-producer: each scoped sub-member writes their own observations *and* receives routed entries from the friction-curator. Intentional architectural choice — curator delivers cross-team observations into the scoped topic; sub-member synthesizes patterns. Documented on team.json's knowledgeTopics comments.

### monetization

**Mission:** own the canonical monetization plan — catalog, tiers, channels, funnel, revenue lines, financial model.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `opportunity-scout` | `opportunity-inbox/*` (taxonomy: monetization-opportunity, classifier: signal-classifier) | `candidate-sku-record/*` | — |
| `market-validator` | `validation-inbox/*` (taxonomy: monetization-validation, classifier: signal-classifier), `monetization-benchmark-adjacent-record/*` (cross-team from marketing) | `monetization-benchmark-record/*` | reads `monetization-benchmark-adjacent-record/*` ← marketing-crew |
| `catalog-strategist` | _(none — proactive; reads Swarm Manager work)_ | `monetization-canon/*` (por_file → `path:docs/monetization/catalogs/CATALOG.md`) | — |
| `financial-tracker` | _(none — proactive)_ | `monetization-ledger-log/*` | — |

**Observations (draft):**
- `candidate-sku-record/*` and `monetization-benchmark-record/*` have no documented consumer. `catalog-strategist` likely *should* consume `candidate-sku-record/*` (it owns catalog promotion); declare it as `intake[].source_team = "monetization"` (same-team) once confirmed. Workshop.
- `financial-tracker` produces a ledger-shaped log (`monetization-ledger-log/*`). No drainer; consumed by operator and by work-flow. Same pattern as `run-lesson-report/*` (see meta-optimization observations).

### scenario-qa

**Mission:** ensure scenario quality through structural quality audits, root-cause bug investigation, and contrarian challenge of QA outcomes. (Pre-emptive readiness ordering moved to swarm-manager's fix-before-feature gate.)

**Plan of record:** [`path:docs/scenario-qa/`](../scenario-qa/) — README, three paired-doc-and-skill method registries (`methods/investigation/`, `methods/audit/`, `methods/readiness/`), and the `taxonomies/bug-report/` taxonomy. Owner-curated like every other team PoR.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `quality-auditor` | _(none — proactive)_ | `quality-audit/*` | — |
| `bug-investigator` | `bug-inbox/*` (taxonomy: `bug-report`, source: `*` — universal-source) | `bug-investigation-report/*` | producers: every team via the `report-bug` writer skill |

**Observations:**
- `bug-inbox/*` is one of the system's two **universal-source intakes** (semantics in `TOPICS_SCHEMA.md` § Universal-source intakes; sister flow is `friction-inbox/*` on meta-optimization). The investigator validates the producer's signal-type assignment as the first sub-step of investigation; deterministic-prefix routing, no separate classifier skill.
- `bug-investigation-report/*` is an audit log, not an inbox. Append-only; one entry per closed bug; drives technique-graduation decisions on `meta-self-improvement`. No drainer; `orphan_output` warning is by-design here.
- **Possible future gap:** `topic[future]:qa-inbox/*` / `topic[future]:audit-inbox/*` for operator-fed "look at this scenario" alpha. No producer today; would `orphan_input`. Documented as future PoR work in `path:docs/scenario-qa/README.md` § Future PoR work; revisit when (e.g.) `vision-walk-prep` adds them as output prefixes.

---

## Validator rules and CI severity

`prompt-manager graph topics` runs every cross-graph rule on every load, splitting findings between **error** severity (CI gate — non-zero exit code) and **warning** severity (advisory). The full rule/severity registry lives with the pillar owners: P1 (declared-graph) rules in [`TOPICS_SCHEMA.md`](TOPICS_SCHEMA.md) § Validation rules; P2 (prose-scan) rules in [`PROSE_SCAN_TARGETS.md`](PROSE_SCAN_TARGETS.md) § Pattern set and § Severity guidance. [`PRIMITIVES.md`](PRIMITIVES.md) § Three Pillars of Topic Validation indexes all three pillars. This section owns only the promotion discipline that spans them.

**Severity flip discipline.** Promoting a rule to error is an explicit, work-item-gated change, not a one-off edit. The contract is: a rule lands at warning, every existing finding is reconciled, then the severity is changed in code AND the rule's home pillar doc (`TOPICS_SCHEMA.md` for P1 rules, `PROSE_SCAN_TARGETS.md` for `prose_topic_leak`). After promotion, *new* findings break CI; reverting to warning to silence drift is forbidden — fix the underlying drift instead.

---

## Adoption checklist

For someone adding a new topic to a member:

1. **Pick the right shape** from § Topic shapes.
2. **Pick a name** that follows § Stable conventions and doesn't collide with an existing topic in § Per-team topic registry.
3. **Declare the producer side** on the producing member's `topics.json#output[]` with `destination_kind`, optional `destination_team`, optional `schema`.
4. **Declare the drainer side** on the drainer's `topics.json#intake[]` with `taxonomy` and (when judgment is required) `classifier_skill`. If no drainer exists yet, file a `capability-work` work item instead — don't ship undrained topics.
5. **Author the taxonomy if needed.** If the new shape doesn't fit any existing taxonomy (`marketing-research`, `monetization-opportunity`, `monetization-validation`, `bug-report`, `friction-report`), author a new taxonomy JSON sidecar + PoR per `INTAKE_PIPELINE.md` § Two routing modes and `README.md` § Active taxonomies.
6. **Run the validator.** `prompt-manager graph topics` must report zero new errors.
7. **Update this file.** Add the topic to § Per-team topic registry under the relevant team.
8. **Declare any new producer structurally.** If the topic introduces a new producer (cross-team flow, scheduled scan, vision-walk input, proactive self-generation), record it in the consuming member's `topics.json` via `external_producers[]` or `intake[].source_team`. A human-readable inputs registry (`INPUTS.md`) is workshop-pending (see `README.md` § Mental Model); until it lands, the structural declaration is the source of truth.
