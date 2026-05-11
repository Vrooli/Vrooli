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

Every topic that has a drainer — any topic declared in some member's `intake[]` — is processed by that member through a **uniform action set**. The drainer reads each entry, applies its taxonomy (and optionally a classifier skill — see `INTAKE_PIPELINE.md` § Two routing modes), and chooses exactly one outcome:

| Outcome | Effect on the entry |
|---|---|
| drop | delete; entry leaves the system |
| observe | retag to a canonical-surface topic (entry's permanent home) |
| promote-to-canon | observe + raise an owned-context decision |
| route-to-another-topic | retag/rewrite under a different topic for a different drainer to handle |
| file-decision | raise an owned-context or capability-gap decision; entry deleted or retained per taxonomy rules |
| file-backlog / file-initiative | hand off to swarm-manager via CLI; entry deleted or retained |

**The action set is uniform across the system.** Whatever produces an entry — operator alpha, vision-walk brainstorm, scheduled scan, cross-team handoff, proactive self-generation, another agent's `route-to-another-topic` outcome — the receiving drainer applies this same set. When the action is `route-to-another-topic`, the receiving drainer applies the set again to the new entry. The architecture is recursive: every entry is processed by *some* drainer through the same outcome set until it terminates in drop / observe / canon / decision / backlog.

This is the load-bearing claim that makes the substrate auditable. Adding a new input shape doesn't require new architecture — only a topic + a drainer + a taxonomy entry. See `INTAKE_PIPELINE.md` § Promotion / Routing for the full table of when each outcome is allowed, and `DECISIONS.md` §4 for the direct-write-vs-swarm-manager threshold.

The drainer's classifier/triage skill (when one exists) is loaded by the heartbeat builder from the member's `intake[].classifier_skill` and lives at `path:scenarios/prompt-manager/store/skills/packs/core/<id>/`. Today's classifiers: `marketing-signal-classifier`, `monetization-signal-classifier`, `market-validation-triage`. Topics with deterministic prefixes (e.g., `notebook-debt`-taxonomy intakes) need no classifier — the prefix segment after the inbox name *is* the signal type.

For **universal-source intakes** (`intake[].source_team = "*"` — any team's members may write; today: `bug-inbox/*` on scenario-qa, `friction-inbox/*` on meta-optimization), the trigger paragraph that tells producers when to invoke the writer skill is rendered into every member's heartbeat prompt via the Storage Map's `## Observe` subsection (`path:scenarios/prompt-manager/api/heartbeat/prompt_builder.go:buildStorageMapSection`). When you add a new universal-source intake, update that section so producers actually receive the trigger — see `TOPICS_SCHEMA.md` § Universal-source intakes for the convention. With two universal observation flows now in place (bugs and friction), the pattern (intake + writer skill + drainer + trigger paragraph) is at the threshold where a data-driven rendering off `intake[].source_team == "*"` declarations becomes worth exploring rather than a third hardcoded paragraph.

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
- `evidence_consumed[]` if the consumer cites entries as evidence on a named decision context.

If no consumer is identified yet, that's a smell: file a `capability-gap` decision or onboard a consumer. Don't let an undrained topic accumulate orphaned entries; the validator will flag it as `orphan_output`. Conversely, if a consumer's `required_read[]` references a prefix nobody outputs, the validator surfaces it as `unread_required`. Both rules read from the same declaration substrate; see `TOPICS_SCHEMA.md` § Validation rules for the full set.

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
    Mechanical constraint: the `<subject>` must NOT itself be a vocabulary word, so `topic[example]:quality-audit/*` is fine (`quality` is the subject, `audit` is the suffix), but `topic[example]:audit-audit/*` is rejected. Picking the right suffix is structural, not aesthetic — `*-audit/*` carries audit-shaped front-matter; `*-scan/*` carries scan-shaped; etc. Audit vs. scan: look at the producing member's `decisions_owned` — `*-finding` / `*-violation` / `*-gap` contexts mean adversarial (audit), pure observation collection means survey (scan). Reports vs. records: a report carries reasoning (a finding + evidence + rationale); a record carries one entity's state (one row, no narrative). Logs are append-only event streams; preps are synthesis briefings explicitly authored for a downstream consumer.
- **Inbox / synthesis topics use purpose suffixes, not type suffixes.** The vocabulary above is for *canonical* surfaces. Transient and synthesis surfaces use a different convention:
    - `*-inbox/*` — transient queue, drained per entry (see § Topic shapes Inbox row)
    - `<short-domain>/notebook/*` — team-scoped curation surface (see § Topic shapes Notebook row)
- **Team name in the prefix only when the team name is part of the *concept*.** Team scoping is implicit — every entry already lives in some team's knowledge store, and cross-team flow is declared explicitly via `destination_team` / `source_team`. Adding a team name to the prefix is therefore redundant by default, and *coupling* the topic to a particular team makes it harder to reorganize (re-assign a topic to a different drainer, move it between teams, or have multiple teams share a concept). The default is **no team in the prefix** — `audience-scan/*`, `competitor-record/*`, `bug-inbox/*`, `qa-run/*`. Add the team only when the team name is genuinely part of the concept (e.g., `marketing-canon/*` and `monetization-canon/*` are two distinct PoR-write surfaces that need to coexist as named concepts), or when the topic is a hierarchical team-internal namespace (e.g., `marketing/notebook/<signal-type>/<slug>` — the notebook IS team-internal and signal types nest under it). When team appears, use the **short domain name** (`marketing`, not `marketing-crew`); the only abbreviation in active use is `marketing-crew` → `marketing` (others — `monetization`, `meta-optimization`, `scenario-qa`, `infra-health`, `director-swarm` — match team id verbatim).

### Topic shapes (current usage)

| Shape | Pattern | Examples | Lifetime |
|---|---|---|---|
| **Inbox (transient)** | `<inbox-name>/<signal-type>/<slug>` | `research-inbox/audience/foo`, `opportunity-inbox/competitor-move/bar`, `bug-inbox/regression/baz`, `friction-inbox/toolchain/qux` | drained → entry retagged or deleted |
| **Notebook debt** | `<short-domain>/notebook/<signal-type>/<slug>` | `marketing/notebook/audience-question/foo`, `meta-optimization/notebook/skill-debt/bar` | drained → promoted, retired, or aged |
| **Scan (survey-shaped observation)** | `<subject>-scan/<slug>` | `audience-scan/foo`, `debt-scan/bar`, `topic[example]:quality-audit/baz` (audit, not scan — see below) | durable; the entry's permanent home |
| **Audit (adversarial / compliance review)** | `<subject>-audit/<slug>` | `topic[example]:quality-audit/foo`, `toolchain-audit/bar`, `runtime-health-audit/baz`, `platform-code-audit/qux`, `skill-audit/quux`, `action-audit/...`, `team-audit/...`, `agent-audit/...` | durable; not retagged |
| **Record (entity record)** | `<subject>-record/<slug>` | `competitor-record/foo`, `hook-record/bar`, `candidate-sku-record/baz`, `monetization-benchmark-record/qux`, `topic[example]:outcome-target-record/quux`, `initiative-portfolio-record/...`, `friction-triage-record/...` | durable; one entity per entry |
| **Draft (in-flight artifact)** | `<subject>-draft/<slug>` | `campaign-draft/foo` | durable until promotion to canon or record; mutable until then |
| **Log (append-only events)** | `<subject>-log/<slug>` | `publish-log/foo`, `monetization-ledger-log/bar`, `qa-run/baz` (run is implicit log; treated as such) | durable, append-only |
| **Canon (PoR edit)** | `<team>-canon/<slug>` | `marketing-canon/foo` (→ docs/marketing/strategy/STRATEGY.md or AUDIENCES.md), `monetization-canon/bar` (→ docs/monetization/catalogs/CATALOG.md) | translates to a PoR markdown edit; `destination_kind = por_file` |
| **Report (investigative finding)** | `<subject>-report/<slug>` | `bug-investigation-report/login-500-2026-05-03`, `friction-report/toolchain/2026-05-03/cli-flag-confusion`, `run-lesson-report/2026-04-25`, `challenge-report/dec-1777060904331053267` | durable; one finding per entry, with rationale |
| **Prep (synthesis briefing)** | `<subject>-prep/<slug>` | `topic[example]:workshop-decision-prep/2026-05-03` | durable; consumed by a downstream synthesizer or operator |
| **Cross-team flow** | same as any of the above, with explicit `destination_team` / `source_team` | `monetization-benchmark-adjacent-record/foo` (marketing → monetization) | depends on the receiving member's drain |

### Notebook vs typed inbox

Both are kinds of intake but serve different roles. A typed inbox (`*-inbox/*`) is for observations whose destination is *known* — the producer assigns a hint signal-type, the drainer classifies and routes promptly, and the inbox view is the unrouted set (entries must leave). A notebook (`<team>/notebook/*`) is for observations whose destination is *not yet known* — the producer just drops the entry, the curator decides later, and entries may persist as `still-debt` indefinitely. The notebook is therefore the residual surface for the catch-all case; as concrete observation types graduate to their own typed inboxes, the notebook narrows to ever-more-residual content.

The producer-side rule that follows from this: if you discover something with a clear typed destination, write to that inbox directly. Use the notebook only when no typed inbox fits. The curator's job includes spotting recurring notebook signal-types and proposing graduation to a typed inbox via `meta-self-improvement` decision.

### Contrarian challenge topics

Contrarian topics are decision-feedback surfaces, not a global operator inbox. The canonical lifecycle lives in [`CONTRARIAN_REVIEW.md`](CONTRARIAN_REVIEW.md).

- `challenge-report/<decision-id>` is an append-only report-shaped topic. It exists only when a contrarian found a concrete failure-mode hit on the target decision.
- `challenge-resolution-record/<decision-id>` is a record-shaped latest-state topic. It is owned by the same contrarian, declares `supersedesPrevious: true`, and records whether the challenge is `open`, `author-responded`, `resolved`, `escalated`, `overridden`, or `stale`.
- Authors do not drain these topics as `intake[]`. They consume them as decision evidence via `evidence_consumed[]` for the contexts they own.
- Vision walk summarizes unresolved or escalated challenge state as part of decision review; it does not scrape `challenge-report/*` as a standalone task queue.

---

## Per-team topic registry

Per team: the topics that team currently produces and drains, with first-principles observations on gaps. "Current state" is derived directly from each member's `topics.json`. "Observations" are draft notes; resolution is workshop-pending unless flagged as a structural error.

### director-swarm

**Mission:** keep Vrooli's initiative portfolio flowing through swarm-manager and surface outcome-driven strategy. Director teams consume decisions (not topics) and produce prep artifacts for downstream consumption.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `outcome-strategist` | _(none — proactive; reads decisions)_ | `outcome-target-record/*` | — |
| `portfolio-manager` | _(none — proactive; reads decisions)_ | `initiative-portfolio-record/*` | — |
| `vision-walk-prep` | _(none — proactive; reads decisions)_ | `vision-walk-record/*`, plus produces into other teams' inboxes (see below) | writes `research-inbox/*` → marketing-crew, `opportunity-inbox/*` → monetization, `validation-inbox/*` → monetization |
| `workshop-decision-prep` | _(none — proactive; reads decisions)_ | `workshop-decision-prep/*` | — |

**Observations (draft):**
- `vision-walk-prep` is the canonical example of a **synthesis pipeline** member: it consumes decision state from other teams and produces (a) its own `vision-walk-record/*` artifact for the operator, and (b) seeded entries directly into other teams' inboxes for those teams to drain after the walk. INPUTS.md will document this pattern as a first-class flow.
- No director-swarm member has `intake[]` — confirms the user's mental model that director teams pull decisions rather than drain topics. This is correct, not a gap.
- `outcome-target-record/*` and `initiative-portfolio-record/*` are consumed by the operator (and possibly by `swarm-manager`) but no `topics.json` declares the consumer. If a future member or scenario consumes them programmatically, declare it via `intake[].source_team = "director-swarm"`.

### infra-health

**Mission:** protect Vrooli's local platform; turn runtime signals + platform-code audits + contrarian review into operator-routed reliability work.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `runtime-health-scanner` | _(none — proactive)_ | `runtime-health-audit/*` | — |
| `platform-code-auditor` | _(none — proactive)_ | `platform-code-audit/*` | — |
| `infra-contrarian` | _(none — proactive; reads peer decisions)_ | `challenge-report/*`, `challenge-resolution-record/*` | — |

**Observations (draft):**
- All three members are pure proactive producers; none have `intake[]`. This is consistent with infra-health's mission (audit-driven, not signal-driven).
- No `topic[future]:infra-inbox/*` exists. If the operator wants to push infra-relevant alpha into this team, no current topic catches it. **Possible gap:** a `topic[future]:infra-inbox/*` for operator-fed runtime/platform alpha, drained by `runtime-health-scanner` or a new triage member. Workshop.
- Contrarian output follows the shared challenge lifecycle in § Contrarian challenge topics.

### marketing-crew

**Mission:** own Vrooli's external voice — subscription marketing, OSS narrative, brand canon, publishing pipeline.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `researcher` | `research-inbox/*` (taxonomy: marketing-research, classifier: marketing-signal-classifier) | `audience-scan/*`, `competitor-record/*`, `hook-record/*`, `monetization-benchmark-adjacent-record/*` | writes `monetization-benchmark-adjacent-record/*` → monetization |
| `brand-manager` | `marketing/notebook/*` (taxonomy: notebook-debt, no classifier) | `marketing-canon/*` (por_file → `path:docs/marketing/strategy/STRATEGY.md`, `path:docs/marketing/strategy/AUDIENCES.md`) | — |
| `publisher` | _(none — proactive)_ | `publish-log/*` | — |
| `oss-advertiser` | _(none — proactive)_ | `campaign-draft/*` | — |
| `subscription-advertiser` | _(none — proactive)_ | `campaign-draft/*` | — |
| `marketing-contrarian` | _(none — proactive; reads peer decisions)_ | `challenge-report/*`, `challenge-resolution-record/*` | — |

**Observations (draft):**
- `campaign-draft/*` is written by both `oss-advertiser` and `subscription-advertiser` but has no declared drainer. `publisher` *should* be the drainer (it owns content publishing) but its `topics.json` doesn't declare `intake: [{ prefix: "campaign-draft/*", ... }]`. **Likely gap:** `publisher` should drain `campaign-draft/*`, picking from drafts and producing publish-log entries. Workshop.
- The two advertisers (`oss-advertiser`, `subscription-advertiser`) are proactive but have no input shape — they generate from operator/vision-walk only. **Possible gap:** a `topic[future]:marketing-brief-inbox/*` (or similar) for cross-team handoff (e.g., monetization → marketing for SKU-launch coverage). Workshop.
- `marketing-canon/*` is unusual: two `output[]` entries on the same prefix with different `destination_path` values (one to STRATEGY.md, one to AUDIENCES.md). This is a known pattern for por_file topics — the prefix names the *category*, the destination_path names the file. Document the pattern explicitly in § Stable conventions if it's the intended design.

### meta-optimization

**Mission:** apply evolutionary pressure to Vrooli's meta-layer — skills, agents, teams, tool contracts.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `debt-curator` | `meta-optimization/notebook/*` (taxonomy: notebook-debt, no classifier) | `debt-scan/*` | — |
| `skill-optimizer` | _(none — proactive)_ | `skill-audit/*`, `action-audit/*` | — |
| `team-agent-optimizer` | _(none — proactive)_ | `team-audit/*`, `agent-audit/*` | — |
| `toolchain-validator` | _(none — proactive)_ | `toolchain-audit/*` | — |
| `run-introspector` | _(none — proactive)_ | `run-lesson-report/*` | — |
| `meta-contrarian` | _(none — proactive; reads peer decisions)_ | `challenge-report/*`, `challenge-resolution-record/*` | — |
| `friction-curator` | `friction-inbox/*` (taxonomy: `friction-report`, source: `*` — universal-source) | `friction-report/<scope>/*` (delivered to scoped sub-member topics), `friction-triage-record/<YYYY-MM-DD>` | producers: every team via the `report-friction` writer skill |

**Observations (draft):**
- Five proactive auditors plus one notebook-debt drainer plus one contrarian plus one friction router. This is the densest team.
- `friction-inbox/*` is the system's second **universal-source intake** (the first is `bug-inbox/*` on scenario-qa). Any team's members may write via the `report-friction` skill (declared as `external_producers`). The friction-curator validates scope, reclassifies `unknown`, and routes by writing to the existing `friction-report/<scope>/<date>/<slug>` topics on the scoped-topic owners' behalf. Curator owns no decision contexts — routing is determinate; capability-gaps are still raised by the scoped-topic owners.
- `friction-triage-record/<YYYY-MM-DD>` is a daily snapshot, supersedesPrevious=true, drained by debt-curator (synthesis input) and by operator review. `orphan_output` warning is by-design here (no peer drainer).
- `run-lesson-report/*` is heavily consumed by `meta-contrarian` (via `decisions_consumed: ["run-lesson", ...]`) and by other optimizers' input gathering, but no `intake[]` declares it. The flow goes through decisions, not direct topic-drain. Document this as the canonical example of "topic → decision-consumer" flow rather than "topic → topic-drainer."
- The four scoped friction topics (`friction-report/toolchain/*`, `friction-report/run-execution/*`, `friction-report/prompt-team-agent-storage/*`, `friction-report/recurring-workaround/*`) are now multi-producer: each scoped sub-member writes their own observations *and* receives routed entries from the friction-curator. Intentional architectural choice — curator delivers cross-team observations into the scoped topic; sub-member synthesizes patterns. Documented on team.json's knowledgeTopics comments.

### monetization

**Mission:** own the canonical monetization plan — catalog, tiers, channels, funnel, revenue lines, financial model.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `opportunity-scout` | `opportunity-inbox/*` (taxonomy: monetization-opportunity, classifier: monetization-signal-classifier) | `candidate-sku-record/*` | — |
| `market-validator` | `validation-inbox/*` (taxonomy: monetization-validation, classifier: market-validation-triage), `monetization-benchmark-adjacent-record/*` (cross-team from marketing) | `monetization-benchmark-record/*` | reads `monetization-benchmark-adjacent-record/*` ← marketing-crew |
| `catalog-strategist` | _(none — proactive; reads decisions)_ | `monetization-canon/*` (por_file → `path:docs/monetization/catalogs/CATALOG.md`) | — |
| `financial-tracker` | _(none — proactive)_ | `monetization-ledger-log/*` | — |
| `monetization-contrarian` | _(none — proactive; reads peer decisions)_ | `challenge-report/*`, `challenge-resolution-record/*` | — |

**Observations (draft):**
- `candidate-sku-record/*` and `monetization-benchmark-record/*` have no documented consumer. `catalog-strategist` likely *should* consume `candidate-sku-record/*` (it owns catalog promotion); declare it as `intake[].source_team = "monetization"` (same-team) once confirmed. Workshop.
- `financial-tracker` produces a ledger-shaped log (`monetization-ledger-log/*`). No drainer; consumed by operator and by decision-flow. Same pattern as `run-lesson-report/*` (see meta-optimization observations).

### scenario-qa

**Mission:** ensure scenario quality through structural quality audits, programmatic readiness reviews, root-cause bug investigation, and contrarian challenge of QA outcomes.

**Plan of record:** [`path:docs/scenario-qa/`](../scenario-qa/) — README, three paired-doc-and-skill registries (`investigation-techniques/`, `audit-techniques/`, `readiness-checks/`), `BUG_REPORT_TAXONOMY.md`. Owner-curated like every other team PoR.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `programmatic-qa-runner` | _(none — proactive)_ | `qa-run/*`, `reviewed-scenario/*`, `dependency-wiring` | — |
| `quality-auditor` | _(none — proactive)_ | `quality-audit/*` | — |
| `bug-investigator` | `bug-inbox/*` (taxonomy: `bug-report`, source: `*` — universal-source) | `bug-investigation-report/*` | producers: every team via the `report-bug` writer skill |
| `qa-contrarian` | _(none — proactive)_ | `challenge-report/*`, `challenge-resolution-record/*` | — |

**Observations:**
- `bug-inbox/*` is one of two **universal-source intakes** in the system; the other is `friction-inbox/*` on meta-optimization. Any team's members may write via the `report-bug` skill (declared as `external_producers`). The investigator validates the producer's signal-type assignment as the first sub-step of investigation; deterministic-prefix routing, no separate classifier skill.
- `bug-investigation-report/*` is an audit log, not an inbox. Append-only; one entry per closed bug; drives technique-graduation decisions on `meta-self-improvement`. No drainer; `orphan_output` warning is by-design here.
- `challenge-report/*` and `challenge-resolution-record/*` follow the shared contrarian challenge lifecycle in § Contrarian challenge topics.
- **Possible future gap:** `topic[future]:qa-inbox/*` / `topic[future]:audit-inbox/*` for operator-fed "look at this scenario" alpha. No producer today; would `orphan_input`. Documented as future PoR work in `path:docs/scenario-qa/README.md` § Future PoR work; revisit when (e.g.) `vision-walk-prep` adds them as output prefixes.

---

## Validator rules and CI severity

`prompt-manager graph topics` runs every cross-graph rule on every load. Rules are split between **error** severity (CI gate — non-zero exit code) and **warning** severity (advisory). The split follows the rule lifecycle pattern: rules land at warning, the existing population of findings is reconciled, and severity is promoted to error in lockstep with code + doc updates so CI catches regressions thereafter.

| Rule | Severity | What it catches |
|---|---|---|
| `conflicting_drain` | error | Two members declare overlapping intake prefixes (would race the drain). |
| `orphan_input` | error | Intake prefix has no producer (no member output, no `external_producers`, no wildcard source_team). |
| `missing_taxonomy` / `unknown_taxonomy` | error | Intake declares no taxonomy or names one that doesn't resolve in the registry. |
| `dangling_por_sink` | error | `destination_kind=por_file` references a `destination_path` that does not exist. |
| `dangling_evidence_decision` | error | `evidence_consumed[].for_decisions[]` references a decision-context id no team's `team.json` declares. |
| `attribution_malformed` | error | Post-cutoff knowledge entry has structurally broken attribution (defense in depth — API rejects this at write time). |
| `unread_required` | error | `required_read[]` prefix has no producer. Producer = any member's `output[]` overlap or any writer-skill `writes_to[]` overlap. |
| `actual_writer_undeclared` (agent-member subcase) | error | A `kind=agent-member` knowledge entry's topic does not overlap that member's declared `output[]`, or the entry claims a member id that doesn't exist on the team. |
| `prose_topic_leak` (cli-knowledge-* subpatterns) | error | Markdown prose contains a `prompt-manager team knowledge-*` invocation (`-add`, `-list`, `-list --topic-prefix`, `-update`) whose topic prefix does not resolve against the relevant declaration set per `PROSE_SCAN_TARGETS.md` § Cross-reference matrix. |
| `actual_writer_undeclared` (external-threshold subcase) | warning | A team's `policy.flagExternalWritesPerWeek` cap was exceeded in some ISO week. Operator-tunable, not concrete drift. |
| `prose_topic_leak` (`marked-topic-ref` / `inferred-backtick-topic-ref` patterns) | warning | A marked topic ref or inferred unmarked backticked topic-shaped string has no matching declaration. Kept at warning permanently because inferred matches intentionally remain a backstop for agent-written docs that omit markers. |
| `topic_key_prefix_mismatch` | warning | Knowledge entry's topic does not match any declared prefix on its team. Surfaces real-data drift; resolved by either adding the declaration or renaming the entry. Not promoted because remediation often spans multiple PRs and the data is real. |
| `orphan_output` | warning | Output prefix has no peer-member consumer. Operator-only snapshots are legitimate (audit logs, ledgers); the warning is the prompt to either add an intake or accept by-design. |
| `missing_destination_schema` | warning | Output names a schema not declared by any taxonomy. Soft signal — frequently a missing taxonomy entry, not drift. |
| `wildcard_source_misuse` | warning | A `source_team=*` intake without an `external_producers` anchor. |

**Severity flip discipline.** Promoting a rule to error is an explicit, decision-gated change, not a one-off edit. The contract is: a rule lands at warning, every existing finding is reconciled, then the severity is changed in code AND the relevant canon doc (this file plus the rule's home doc, e.g., `PROSE_SCAN_TARGETS.md` for `prose_topic_leak`). After promotion, *new* findings break CI; reverting to warning to silence drift is forbidden — fix the underlying drift instead.

**Why writer-skill consultation is part of `unread_required`.** Writer-skill `writes_to[]` is the producer-side declaration for skill-written prefixes. A required_read prefix that overlaps a writer-skill's writes_to[] has a documented producer; demanding a member-side output[] in addition would force false declarations (e.g., friction-curator does not write `friction-inbox/*`; the report-friction skill does). The rule consults both sources.

**Why the prose scanner has read/write split.** Writer skills legitimately read other teams' topics (queue depth, source data) — those references must resolve against any team's declaration set, not the skill's own writes_to[]. Only `knowledge-add` / `knowledge-update` (write patterns) require writes_to[] coverage. See `PROSE_SCAN_TARGETS.md` § Cross-reference matrix and `prose_scan.go::joinProseMatch::proseTargetSkill`.

---

## Adoption checklist

For someone adding a new topic to a member:

1. **Pick the right shape** from § Topic shapes.
2. **Pick a name** that follows § Stable conventions and doesn't collide with an existing topic in § Per-team topic registry.
3. **Declare the producer side** on the producing member's `topics.json#output[]` with `destination_kind`, optional `destination_team`, optional `schema`.
4. **Declare the drainer side** on the drainer's `topics.json#intake[]` with `taxonomy` and (when judgment is required) `classifier_skill`. If no drainer exists yet, file a `capability-gap` decision instead — don't ship undrained topics.
5. **Author the taxonomy if needed.** If the new shape doesn't fit any existing taxonomy (`marketing-research`, `monetization-opportunity`, `monetization-validation`, `notebook-debt`), author a new taxonomy JSON sidecar + PoR per `INTAKE_PIPELINE.md` § Two routing modes and `README.md` § Active taxonomies.
6. **Run the validator.** `prompt-manager graph topics` must report zero new errors.
7. **Update this file.** Add the topic to § Per-team topic registry under the relevant team.
8. **Update INPUTS.md.** If the topic introduces a new producer (cross-team flow, scheduled scan, vision-walk input, proactive self-generation), declare the source there.
