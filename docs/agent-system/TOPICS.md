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
- **Not the same as the `api/topics/` package's "topics."** That package serves a different concept (parent/child content-taxonomy trees with attached skills) and predates the inbox-flow data layer. Naming collision; different concern. See `README.md` § Naming note.

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

The drainer's classifier/triage skill (when one exists) is loaded by the heartbeat builder from the member's `intake[].classifier_skill` and lives at `scenarios/prompt-manager/store/skills/packs/core/<id>/`. Today's classifiers: `marketing-signal-classifier`, `monetization-signal-classifier`, `market-validation-triage`. Topics with deterministic prefixes (e.g., `notebook-debt`-taxonomy intakes) need no classifier — the prefix segment after the inbox name *is* the signal type.

For **universal-source intakes** (`intake[].source_team = "*"` — any team's members may write; today: `bug-inbox/*` on scenario-qa, `friction-inbox/*` on meta-optimization), the trigger paragraph that tells producers when to invoke the writer skill is rendered into every member's heartbeat prompt via the Storage Map's `## Observe` subsection (`scenarios/prompt-manager/api/heartbeat/prompt_builder.go:buildStorageMapSection`). When you add a new universal-source intake, update that section so producers actually receive the trigger — see `TOPICS_SCHEMA.md` § Universal-source intakes for the convention. With two universal observation flows now in place (bugs and friction), the pattern (intake + writer skill + drainer + trigger paragraph) is at the threshold where a data-driven rendering off `intake[].source_team == "*"` declarations becomes worth exploring rather than a third hardcoded paragraph.

## When to create a topic

Create a new topic when **all three** of these are true:

1. A member needs to declare structurally that it produces or consumes a kind of work.
2. That kind of work doesn't already match an existing topic prefix. When in doubt, run `prompt-manager team knowledge-list <team> --topic-prefix=<candidate>` and skim what's there.
3. The work is **signal-shaped** — discrete entries that go through a lifecycle, not ad-hoc text or per-task chatter.

Do **not** create a topic for:

- One-off communication between agents (use direct member-to-member channels or the team-inbox messaging system, which is a different surface — see `PRIMITIVES.md`).
- Configuration or static documentation (PoR or member files).
- Pure logging without intent to drain (scenario logs or telemetry).
- Parallel naming for an existing topic (e.g., creating `competitor-scan/*` when `competitor/*` already exists). Naming inconsistencies that survive go in § Naming conventions to be reconciled, not minted as new topics.

When you do create one: declare it in the producer's `topics.json#output[]` and the drainer's `topics.json#intake[]` — both sides. If no drainer is identified yet, that's a smell: file a `capability-gap` decision or onboard a new drainer. Don't let an undrained topic accumulate orphaned entries; the validator will flag it as `orphan_output`.

---

## Naming conventions

Topics use kebab-case, are scoped to a team's knowledge store unless cross-team flow is declared, and pick a prefix shape based on what the entries are *for*. The conventions below describe what's in active use; some are stable, some surface known inconsistencies that meta-optimization should resolve via decision.

### Stable conventions

- **kebab-case.** Lowercase ASCII letters, digits, and hyphens. No underscores, no camelCase.
- **Wildcard.** `<prefix>/*` matches any entry whose key starts with `<prefix>/`. Bare `*` is disallowed by the validator.
- **Slug last.** Every full entry key ends in a unique slug under the prefix (e.g., `audience-scan/<slug>`). Slugs are short, kebab-case, descriptive.
- **One topic, one purpose.** A topic prefix is for one kind of work. Don't co-mingle audience observations and competitor pricing under a single prefix because they're in the same domain — they're different signal types and need different routing.
- **Slash-slug entry keys.** Every knowledge entry's `topic` field is `<declared-prefix>/<slug>` — the prefix matches some member's `topics.json` declaration with any trailing `/*` stripped, the slug names the entry under that prefix. Date-rotated snapshots use the date as the slug (`audience-scan/2026-04-23`); per-case reports use a descriptive slug (`bug-investigation/login-500-2026-05-03`). Flat-hyphenated keys (`audience-scan-2026-04-23`) and bare tokens with no slash structure are forbidden — they don't match prefix-based queries (`prompt-manager team knowledge-list <team> --topic-prefix=<prefix>/` returns nothing for a flat key). The `topic_key_prefix_mismatch` validation rule (severity warning) flags any entry whose topic doesn't match a declared prefix on its team.
- **`-audit` vs `-scan` suffix.** Adversarial / compliance topics — those whose owned-decision contexts are findings, violations, or other "X is broken against a standard" shapes — use `-audit`. Survey / observation topics — those whose entries are just observations of a domain without an implicit standard — use `-scan`. Mechanical test: look at the producing member's `decisions_owned` in `topics.json`; if any context is named `*-finding`, `*-violation`, `*-gap` against a standard, it's audit. Examples: `platform-code-audit/*`, `team-audit/*`, `toolchain-audit/*` are adversarial; `audience-scan/*`, `debt-scan/*` are survey. The suffix is optional only when the bare domain name already unambiguously names the surface (e.g., `competitor/*`, `hook/*` are durable observation surfaces with no `-scan` suffix); when in doubt, add the suffix.
- **Team name in the prefix only when the team name is part of the *concept*.** Team scoping is implicit — every entry already lives in some team's knowledge store, and cross-team flow is declared explicitly via `destination_team` / `source_team`. Adding a team name to the prefix is therefore redundant by default, and *coupling* the topic to a particular team makes it harder to reorganize (re-assign a topic to a different drainer, move it between teams, or have multiple teams share a concept). The default is **no team in the prefix** — `audience-scan/*`, `competitor/*`, `bug-inbox/*`, `qa-run/*`. Add the team only when the team name is genuinely part of the concept (e.g., `marketing-canon/*` and `monetization-canon/*` are two distinct PoR-write surfaces that need to coexist as named concepts), or when the topic is a hierarchical team-internal namespace (e.g., `marketing/notebook/<signal-type>/<slug>` — the notebook IS team-internal and signal types nest under it). When team appears, use the **short domain name** (`marketing`, not `marketing-crew`); the only abbreviation in active use is `marketing-crew` → `marketing` (others — `monetization`, `meta-optimization`, `scenario-qa`, `infra-health`, `director-swarm` — match team id verbatim).

### Topic shapes (current usage)

| Shape | Pattern | Examples | Lifetime |
|---|---|---|---|
| **Inbox (transient)** | `<inbox-name>/<signal-type>/<slug>` | `research-inbox/audience/foo`, `opportunity-inbox/competitor-move/bar` | drained → entry retagged or deleted |
| **Notebook debt** | `<short-domain>/notebook/<signal-type>/<slug>` | `marketing/notebook/audience-question/foo`, `meta-optimization/notebook/skill-debt/bar` | drained → promoted, retired, or aged |
| **Canonical surface (knowledge)** | `<surface>/<slug>` | `audience-scan/foo`, `competitor/bar`, `hook/baz`, `candidate-sku/qux` | durable; the entry's permanent home |
| **Cross-team flow** | same as inbox or canonical, with explicit `destination_team` / `source_team` | `monetization-benchmark-adjacent/foo` (marketing → monetization) | depends on the receiving member's drain |
| **Audit output** | `<purpose>-audit/<slug>` | `quality-audit/foo`, `toolchain-audit/bar`, `runtime-health-audit/baz` | adversarial / compliance findings; durable, not retagged |
| **Scan output** | `<purpose>-scan/<slug>` | `audience-scan/foo`, `debt-scan/bar` | survey / observation collection; durable, not retagged |
| **PoR-write topic** | `<team>-canon/<slug>` | `marketing-canon/foo`, `monetization-canon/bar` | translates to a PoR markdown edit; `destination_kind = por_file` |
| **Decision-prep / synthesis** | `<purpose>/<slug>` | `vision-walk-prep/foo`, `workshop-decision-prep/bar`, `initiative-portfolio/baz`, `outcome-targets/qux` | durable; consumed by another member's intake or by the operator |
| **Local audit log** | `<purpose>/<slug>` | `publish-log/foo`, `qa-run/bar`, `run-lessons/baz`, `debt-scan/qux`, `monetization-ledger/quux` | durable; usually not drained, just queryable |

### Notebook vs typed inbox

Both are kinds of intake but serve different roles. A typed inbox (`*-inbox/*`) is for observations whose destination is *known* — the producer assigns a hint signal-type, the drainer classifies and routes promptly, and the inbox view is the unrouted set (entries must leave). A notebook (`<team>/notebook/*`) is for observations whose destination is *not yet known* — the producer just drops the entry, the curator decides later, and entries may persist as `still-debt` indefinitely. The notebook is therefore the residual surface for the catch-all case; as concrete observation types graduate to their own typed inboxes, the notebook narrows to ever-more-residual content.

The producer-side rule that follows from this: if you discover something with a clear typed destination, write to that inbox directly. Use the notebook only when no typed inbox fits. The curator's job includes spotting recurring notebook signal-types and proposing graduation to a typed inbox via `meta-self-improvement` decision.

### Known inconsistencies

These are real today; they should be reconciled via meta-optimization decisions before the registry below is treated as steady-state.

1. ~~**Inbox-naming drift.**~~ **Resolved 2026-05-03.** Standardized on `*-inbox/*` for external/cross-team intake; `validation-queue/*` was renamed to `validation-inbox/*`. `<team>/notebook/*` stays as-is — it's a team-scoped curation surface (notebook-debt taxonomy), not a producer→consumer intake, so the different shape is intentional.
2. ~~**Audit vs. scan suffix.**~~ **Resolved 2026-05-03.** Rule landed in § Stable conventions: `-audit` for adversarial / compliance-shaped findings, `-scan` for survey / observation collection. Two renames executed: `toolchain-scan/*` → `toolchain-audit/*` (toolchain-validator produces `toolchain-violation` decisions — adversarial) and `runtime-health/*` → `runtime-health-audit/*` (runtime-health-scanner produces `runtime-health-finding` decisions — adversarial). The `TOOLCHAIN_SCAN.md` shared snapshot was renamed to `TOOLCHAIN_AUDIT.md` for parity with sibling files.
3. ~~**Slash-slug vs flat-hyphenated entry keys.**~~ **Resolved 2026-05-03.** Rule landed in § Stable conventions: every entry uses `<prefix>/<slug>` form, no flat-hyphenated keys (`<prefix>-<slug>`). 21 `team.json` declarations migrated to slash form, 60+ live knowledge entries renamed across 5 teams (marketing-crew, meta-optimization, monetization, scenario-qa, director-swarm), 6 HEARTBEAT.md files updated, 2 prefix mismatches fixed during migration (`runtime-health` → `runtime-health-audit`, `platform-audit` → `platform-code-audit` — both align team.json with declared topics.json prefixes), 12 atypical legacy entries cleaned up (Portfolio entries → `portfolio-snapshot/<date>`, scenario-qa daily logs → `qa-run/<topic>` or `quality-audit/<topic>`, `dev-log-narrative-principles` → `principles/dev-log-narrative`, `State persistence issue …` → `legacy-note/state-persistence-resolved`). New validator rule `topic_key_prefix_mismatch` (warning) cross-checks each entry's topic against declared prefixes per team — running it now will surface the next layer of gaps (entries under prefixes not yet declared in any `topics.json` output, e.g., `vision-walk/*`, `agent-visited/*`, `principles/*`, `legacy-note/*`, plus snapshot prefixes like `brand-snapshot/*` and `portfolio-snapshot/*` that need `topics.json` declarations or rename). Those become follow-up decisions.
4. **`challenge-note/*` is shared by five contrarians (one per team) but is purely team-local.** Every team's contrarian writes `challenge-note/*` into its own knowledge store; nobody drains it; no `destination_team` is declared. The shared name is fine (it's a *kind* of topic, not a shared surface), but it has no documented consumer. **Recommendation:** either add a drainer (e.g., `meta-contrarian` consumes peer-team `challenge-note/*` cross-team) or document explicitly that `challenge-note/*` is operator-read-only and should be flagged as `orphan_output: warning, by-design`.
5. ~~**Domain-prefix vs team-prefix on canon surfaces.**~~ **Resolved 2026-05-03.** Rule landed in § Stable conventions: team name appears in the prefix only when the team name is part of the concept (e.g., distinct PoR-write surfaces `marketing-canon/*` vs `monetization-canon/*`) or when the topic is a hierarchical team-internal namespace (e.g., `marketing/notebook/*`). Default is no team prefix. The only abbreviation enforced is `marketing-crew` → `marketing` in topic prefixes; rename of `marketing-crew/notebook/*` → `marketing/notebook/*` landed at the same time.

---

## Per-team topic registry

Per team: the topics that team currently produces and drains, with first-principles observations on gaps. "Current state" is derived directly from each member's `topics.json`. "Observations" are draft notes; resolution is workshop-pending unless flagged as a structural error.

### director-swarm

**Mission:** keep Vrooli's initiative portfolio flowing through swarm-manager and surface outcome-driven strategy. Director teams consume decisions (not topics) and produce prep artifacts for downstream consumption.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `outcome-strategist` | _(none — proactive; reads decisions)_ | `outcome-targets/*` | — |
| `portfolio-manager` | _(none — proactive; reads decisions)_ | `initiative-portfolio/*` | — |
| `vision-walk-prep` | _(none — proactive; reads decisions)_ | `vision-walk-prep/*`, plus produces into other teams' inboxes (see below) | writes `research-inbox/*` → marketing-crew, `opportunity-inbox/*` → monetization, `validation-inbox/*` → monetization |
| `workshop-decision-prep` | _(none — proactive; reads decisions)_ | `workshop-decision-prep/*` | — |

**Observations (draft):**
- `vision-walk-prep` is the canonical example of a **synthesis pipeline** member: it consumes decision state from other teams and produces (a) its own `vision-walk-prep/*` artifact for the operator, and (b) seeded entries directly into other teams' inboxes for those teams to drain after the walk. INPUTS.md will document this pattern as a first-class flow.
- No director-swarm member has `intake[]` — confirms the user's mental model that director teams pull decisions rather than drain topics. This is correct, not a gap.
- `outcome-targets/*` and `initiative-portfolio/*` are consumed by the operator (and possibly by `swarm-manager`) but no `topics.json` declares the consumer. If a future member or scenario consumes them programmatically, declare it via `intake[].source_team = "director-swarm"`.

### infra-health

**Mission:** protect Vrooli's local platform; turn runtime signals + platform-code audits + contrarian review into operator-routed reliability work.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `runtime-health-scanner` | _(none — proactive)_ | `runtime-health-audit/*` | — |
| `platform-code-auditor` | _(none — proactive)_ | `platform-code-audit/*` | — |
| `infra-contrarian` | _(none — proactive; reads peer decisions)_ | `challenge-note/*` | — |

**Observations (draft):**
- All three members are pure proactive producers; none have `intake[]`. This is consistent with infra-health's mission (audit-driven, not signal-driven).
- No `infra-inbox/*` exists. If the operator wants to push infra-relevant alpha into this team, no current topic catches it. **Possible gap:** an `infra-inbox/*` for operator-fed runtime/platform alpha, drained by `runtime-health-scanner` or a new triage member. Workshop.
- `challenge-note/*` orphan-output applies here (see § Known inconsistencies #3).

### marketing-crew

**Mission:** own Vrooli's external voice — subscription marketing, OSS narrative, brand canon, publishing pipeline.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `researcher` | `research-inbox/*` (taxonomy: marketing-research, classifier: marketing-signal-classifier) | `audience-scan/*`, `competitor/*`, `hook/*`, `monetization-benchmark-adjacent/*` | writes `monetization-benchmark-adjacent/*` → monetization |
| `brand-manager` | `marketing/notebook/*` (taxonomy: notebook-debt, no classifier) | `marketing-canon/*` (por_file → `docs/marketing/STRATEGY.md`, `docs/marketing/AUDIENCES.md`) | — |
| `publisher` | _(none — proactive)_ | `publish-log/*` | — |
| `oss-advertiser` | _(none — proactive)_ | `campaign-drafts/*` | — |
| `subscription-advertiser` | _(none — proactive)_ | `campaign-drafts/*` | — |
| `marketing-contrarian` | _(none — proactive; reads peer decisions)_ | `challenge-note/*` | — |

**Observations (draft):**
- `campaign-drafts/*` is written by both `oss-advertiser` and `subscription-advertiser` but has no declared drainer. `publisher` *should* be the drainer (it owns content publishing) but its `topics.json` doesn't declare `intake: [{ prefix: "campaign-drafts/*", ... }]`. **Likely gap:** `publisher` should drain `campaign-drafts/*`, picking from drafts and producing publish-log entries. Workshop.
- The two advertisers (`oss-advertiser`, `subscription-advertiser`) are proactive but have no input shape — they generate from operator/vision-walk only. **Possible gap:** a `marketing-brief/*` inbox or similar for cross-team handoff (e.g., monetization → marketing for SKU-launch coverage). Workshop.
- `marketing-canon/*` is unusual: two `output[]` entries on the same prefix with different `destination_path` values (one to STRATEGY.md, one to AUDIENCES.md). This is a known pattern for por_file topics — the prefix names the *category*, the destination_path names the file. Document the pattern explicitly in § Stable conventions if it's the intended design.

### meta-optimization

**Mission:** apply evolutionary pressure to Vrooli's meta-layer — skills, agents, teams, tool contracts.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `debt-curator` | `meta-optimization/notebook/*` (taxonomy: notebook-debt, no classifier) | `debt-scan/*` | — |
| `skill-optimizer` | _(none — proactive)_ | `skill-audit/*`, `action-audit/*` | — |
| `team-agent-optimizer` | _(none — proactive)_ | `team-audit/*`, `agent-audit/*` | — |
| `toolchain-validator` | _(none — proactive)_ | `toolchain-audit/*` | — |
| `run-introspector` | _(none — proactive)_ | `run-lessons/*` | — |
| `meta-contrarian` | _(none — proactive; reads peer decisions)_ | `challenge-note/*` | — |
| `friction-curator` | `friction-inbox/*` (taxonomy: `friction-report`, source: `*` — universal-source) | `friction/<scope>/*` (delivered to scoped sub-member topics), `friction-triage/<YYYY-MM-DD>` | producers: every team via the `report-friction` writer skill |

**Observations (draft):**
- Five proactive auditors plus one notebook-debt drainer plus one contrarian plus one friction router. This is the densest team.
- `friction-inbox/*` is the system's second **universal-source intake** (the first is `bug-inbox/*` on scenario-qa). Any team's members may write via the `report-friction` skill (declared as `external_producers`). The friction-curator validates scope, reclassifies `unknown`, and routes by writing to the existing `friction/<scope>/<date>/<slug>` topics on the scoped-topic owners' behalf. Curator owns no decision contexts — routing is determinate; capability-gaps are still raised by the scoped-topic owners.
- `friction-triage/<YYYY-MM-DD>` is a daily snapshot, supersedesPrevious=true, drained by debt-curator (synthesis input) and by operator review. `orphan_output` warning is by-design here (no peer drainer).
- `*-audit/*` (skill, action, team, agent) vs `*-scan/*` (toolchain, debt) inconsistency — see § Known inconsistencies #2.
- `run-lessons/*` is heavily consumed by `meta-contrarian` (via `decisions_consumed: ["run-lesson", ...]`) and by other optimizers' input gathering, but no `intake[]` declares it. The flow goes through decisions, not direct topic-drain. Document this as the canonical example of "topic → decision-consumer" flow rather than "topic → topic-drainer."
- The four scoped friction topics (`friction/toolchain/*`, `friction/run-execution/*`, `friction/prompt-team-agent-storage/*`, `friction/recurring-workaround/*`) are now multi-producer: each scoped sub-member writes their own observations *and* receives routed entries from the friction-curator. Intentional architectural choice — curator delivers cross-team observations into the scoped topic; sub-member synthesizes patterns. Documented on team.json's knowledgeTopics comments.

### monetization

**Mission:** own the canonical monetization plan — catalog, tiers, channels, funnel, revenue lines, financial model.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `opportunity-scout` | `opportunity-inbox/*` (taxonomy: monetization-opportunity, classifier: monetization-signal-classifier) | `candidate-sku/*` | — |
| `market-validator` | `validation-inbox/*` (taxonomy: monetization-validation, classifier: market-validation-triage), `monetization-benchmark-adjacent/*` (cross-team from marketing) | `monetization-benchmark/*` | reads `monetization-benchmark-adjacent/*` ← marketing-crew |
| `catalog-strategist` | _(none — proactive; reads decisions)_ | `monetization-canon/*` (por_file → `docs/monetization/CATALOG.md`) | — |
| `financial-tracker` | _(none — proactive)_ | `monetization-ledger/*` | — |
| `contrarian` | _(none — proactive; reads peer decisions)_ | `challenge-note/*` | — |

**Observations (draft):**
- `candidate-sku/*` and `monetization-benchmark/*` have no documented consumer. `catalog-strategist` likely *should* consume `candidate-sku/*` (it owns catalog promotion); declare it as `intake[].source_team = "monetization"` (same-team) once confirmed. Workshop.
- `financial-tracker` produces a ledger-shaped log (`monetization-ledger/*`). No drainer; consumed by operator and by decision-flow. Same pattern as `run-lessons/*` (see meta-optimization observations).

### scenario-qa

**Mission:** ensure scenario quality through deep architectural audits, programmatic readiness reviews, root-cause bug investigation, and contrarian challenge of QA outcomes.

**Plan of record:** [`docs/scenario-qa/`](../scenario-qa/) — README, three paired-doc-and-skill registries (`investigation-techniques/`, `audit-techniques/`, `readiness-checks/`), `BUG_REPORT_TAXONOMY.md`. Owner-curated like every other team PoR.

| Member | Drains (intake) | Writes (output) | Cross-team |
|---|---|---|---|
| `programmatic-qa-runner` | _(none — proactive)_ | `qa-run/*`, `reviewed-scenario/*`, `dependency-wiring` | — |
| `quality-auditor` | _(none — proactive)_ | `quality-audit/*`, `deep-audit/*` | — |
| `bug-investigator` | `bug-inbox/*` (taxonomy: `bug-report`, source: `*` — universal-source) | `bug-investigation/*` | producers: every team via the `report-bug` writer skill |
| `qa-contrarian` | _(none — proactive)_ | `challenge-note/*` | — |

**Observations:**
- `bug-inbox/*` is one of two **universal-source intakes** in the system; the other is `friction-inbox/*` on meta-optimization. Any team's members may write via the `report-bug` skill (declared as `external_producers`). The investigator validates the producer's signal-type assignment as the first sub-step of investigation; deterministic-prefix routing, no separate classifier skill.
- `bug-investigation/*` is an audit log, not an inbox. Append-only; one entry per closed bug; drives technique-graduation decisions on `meta-self-improvement`. No drainer; `orphan_output` warning is by-design here.
- `challenge-note/*` shares the cross-team contrarian-orphan pattern with `marketing-crew`, `monetization`, `meta-optimization`, and `infra-health` (see § Known inconsistencies #3). Workshop-pending.
- **Possible future gap:** `qa-inbox/*` / `audit-inbox/*` for operator-fed "look at this scenario" alpha. No producer today; would `orphan_input`. Documented as future PoR work in `docs/scenario-qa/README.md` § Future PoR work; revisit when (e.g.) `vision-walk-prep` adds them as output prefixes.

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
