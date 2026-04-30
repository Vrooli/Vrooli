# Monetization Team

## Mission
Own the canonical monetization plan for Vrooli — catalog, tiers, channels, funnel, revenue lines, and financial model — and surface the decisions the operator must make each heartbeat to stay on a path to default-alive. The operator is the real strategist; this team maintains the plan, tracks current state against it, and converts measurable changes into concrete decisions.

## Coordination Pattern
Leaderless / independent. Five members, each with its own heartbeat and its own decision stream. There is no AI lead — do not recreate one implicitly through "synthesize the other agents" behavior. Coordination happens outside the team, in the morning vision walk, where the operator reviews pending decisions across all members.

If a member is tempted to aggregate other members' outputs into a single brief, that is the leader-led antipattern. Each member stays in its own lane and produces its own first-class output.

## Members
- **catalog-strategist** — maintains the SKU / tier / channel / scenario graph and proposes promotions when triggers fire.
- **opportunity-scout** — generates and classifies candidate SKUs, add-ons, services lines, and discovery channels.
- **financial-tracker** — maintains the ledger (costs, revenue, channel attribution, time allocation, runway, default-alive gap).
- **market-validator** — grounds pricing and retention assumptions in external benchmarks for the active tier.
- **contrarian** — mandatory skeptic across all other members' outputs and pending decisions.

Each member has an AGENTS.md, SOUL.md, TOOLS.md under `store/agents/<member>/` and a RESPONSIBILITIES.md + HEARTBEAT.md under `store/teams/monetization/members/<member>/`.

## Operating Rules

1. **Default focus is `active` SKUs and the active tier.** Candidate SKUs and candidate tiers are read only to check their revisit triggers. Do not re-evaluate dormant candidates every heartbeat.
2. **No candidate without an explicit revisit trigger.** When opportunity-scout captures a candidate or catalog-strategist promotes an idea to candidate, a concrete trigger condition must be attached. Vibes are not triggers.
3. **Agents propose doc edits via decisions. The operator curates canonical docs.** No member writes directly to the files in `docs/monetization/`. Doc changes happen through decisions with contexts listed below. **Carve-out:** [`shared/operator-inputs.json`](operator-inputs.json) is operator *state*, not team-curated canon. The operator edits it directly (not via decisions); `financial-tracker` reads it to source cash, burn categories, time allocation, and services data. Gathering guidance lives in [`docs/monetization/HOW_TO_GATHER_INPUTS.md`](../../../../../../docs/monetization/HOW_TO_GATHER_INPUTS.md).
4. **Honesty flags are mandatory on every number.** Every metric emitted is labeled `fixed`, `estimate`, `aspirational`, `measured`, or `pending-telemetry`. Unlabeled numbers are guardrail violations.
5. **Open-source self-host is strategic positioning.** Frame the subscription as convenience + integrated gateway. Do NOT frame it as paywalling core features. If an output reads as "users who go free are leaking revenue," the framing is broken.
6. **Agents are the expansion engine.** When proposing acquisition, activation, upsell, or retention mechanisms, default to agent-driven surfaces. Fall back to marketing/email/pop-ups only when agents cannot reach the relevant moment.
7. **Services are a deliberate lever, not a business.** Scenarios are double-revenue assets — sold as products AND operated by us for paying clients (the same shovels we sell are the shovels we use to dig for gold). Services are expected to activate in the post-bundle / pre-default-alive window and produce meaningful revenue, but every active line must have a hypothesis, a fixed-duration pilot, a productization target, and a sunset/convert clause. Services revenue is tracked separately from subscription revenue. The discipline exists because we intend to lean into this lever — not to suppress it.
8. **Discovery channels are not revenue lines.** Channels explain where users or agents come from; revenue lines explain how money flows. Skill registries, OSS discovery, and app stores can validate capability and feed subscription without being monetized directly.
9. **Channel activation requires evidence.** A candidate channel activates only when its channel-specific trigger fires and telemetry can attribute lift. For skill registries, standalone installability, signed/scanned registry publication, and per-skill telemetry are prerequisites.
10. **Tier 4 (hardware) is north-star.** Do not plan work against it without explicit operator initiation.
11. **Every scenario proposal articulates both acquisition AND retention impact.** Scenarios evaluated only on acquisition appeal will starve the retention side of the funnel.
12. **Activation work is retention work.** Most churn is failed activation. When the team proposes retention investments, check whether the real issue is activation first.
13. **Downgrade-to-free is not churn.** Track separately. Different signal, different causes.
14. **Legal surface check on every services line.** Lead-gen (TCPA/CAN-SPAM/GDPR), consulting (contract/IP), done-for-you builds (warranty/liability) all carry distinct regulatory exposure.
15. **Services capacity ≤ 30% of time budget.** Exceeding for 3+ consecutive weeks triggers a services-trap review.
16. **Pre-launch metrics are aspirational.** Do not hallucinate current-state numbers for unmeasured metrics. Use `pending-telemetry` and point at [`TELEMETRY_ROADMAP.md`](../../../../../../docs/monetization/TELEMETRY_ROADMAP.md) instead.
17. **This team does not build telemetry scenarios.** Capability gaps are captured in TELEMETRY_ROADMAP.md; scenarios are built (or not) by feature/QA/portfolio lanes when their priority justifies it.

## Decision Contexts
Members surface decisions with these contexts. The operator reviews them at the morning vision walk.

- `catalog-promotion` — scenario gains headliner status, candidate SKU or tier gets trigger-met and is proposed for promotion
- `catalog-mapping-update` — scenario-sku-map.json change proposal
- `channel-activation` — candidate discovery channel's activation trigger has fired; propose promotion
- `channel-attribution-gap` — active or pilot channel lacks required attribution telemetry
- `sku-retirement` — proposal to retire a SKU or tier
- `services-activation` — candidate services line's trigger has fired; propose promotion
- `services-conversion` — service client → subscription conversion proposal
- `services-sunset` — services line missed productization target; propose sunset
- `pricing-decision` — price setting or adjustment for a specific SKU × tier
- `financial-model-assumption-update` — a load-bearing assumption in the model needs revision
- `runway-warning` — material change in runway or default-alive gap
- `services-trap-warning` — services capacity or services-to-subs ratio exceeded guardrail
- `benchmark-update` — new or refreshed market benchmark
- `funnel-bottleneck` — a funnel stage is identified as the current bottleneck
- `retention-concern` — a retention metric materially worse than target
- `decision-rejection-proposed` — contrarian formally recommends rejecting or revising a pending proposal after it fails multiple failure modes
- `framework-update` — contrarian identifies a real flaw not covered by the existing seven failure modes and proposes updating the framework

Keep decision descriptions short, concrete, and tied to a specific action the operator can take or defer.

## Decision Queue Discipline

The monetization team produces decisions into a single queue the operator reviews at the morning vision walk. The following rules keep the queue sized to the operator's actual review rate, not the agents' emission rate.

### Supersession over stacking (mandatory)

Before any member creates a new pending decision, it **must** check existing pending decisions in its owned context list. If a pending decision is obsolete or redundant with a fresher take, the member:

1. Marks the prior decision `superseded`
2. Creates the new decision with a `supersedes: <prior-decision-id>` reference
3. Does **not** stack a second decision on the same underlying question

Stacking (creating a new decision alongside a superseded-in-spirit prior one) is a guardrail violation. This matches the director-swarm pattern visible in its handoff-history (`"Supersedes dec-..."`).

### Per-member context enumeration

Each member's stop-early thresholds are computed against an explicit context list, not a fuzzy "my contexts" reference:

- **catalog-strategist:** `catalog-promotion`, `catalog-mapping-update`, `channel-activation`, `sku-retirement`, `services-activation`, `services-conversion`, `services-sunset`
- **opportunity-scout:** `catalog-promotion`, `channel-activation` (only via direct promotion; opportunity-pool entries in `opportunities.jsonl` are not decisions)
- **financial-tracker:** `runway-warning`, `services-trap-warning`, `channel-attribution-gap`, `pricing-decision`, `financial-model-assumption-update`, `funnel-bottleneck`, `retention-concern`
- **market-validator:** `benchmark-update`, `pricing-decision`, `financial-model-assumption-update`
- **contrarian:** `decision-rejection-proposed`, `framework-update`

Overlaps (e.g., `pricing-decision` is owned by both financial-tracker and market-validator) are expected. Each member only counts its owned contexts when evaluating its own stop-early threshold.

Services-* contexts live with `catalog-strategist` because services activation / conversion / sunset are SKU-adjacent lifecycle transitions (catalog-strategist already owns SKU lifecycle). Opportunity-scout surfaces services-line candidates in `opportunities.jsonl`; catalog-strategist promotes them. Financial-tracker owns funnel-bottleneck and retention-concern because both surface from the same metric-delta analysis the tracker already performs — and they ride the same `pending-telemetry` flags while funnel data is unmeasured.

### Team-level ceiling

**If total pending monetization decisions exceed 12, all members shift to read-only mode.** Every member's heartbeat, before doing anything else, queries `prompt-manager team decision-list monetization --status=pending --json` and counts the result. If the count is ≥12, the member:

- Skips new-decision creation entirely this heartbeat
- Still writes its knowledge snapshot (ledger entry, catalog snapshot, scout scan, market scan, etc.)
- Still performs supersession if it can collapse any existing pending decisions (supersession shrinks the queue; it's the only decision-write allowed in read-only mode)
- Reports in its handoff: *"Team queue at capacity ([count] pending). Read-only mode this heartbeat."*

12 is a starting number tuned for a ~3/day operator review rate. Revisit after observing real flow.

### Aging policy

A pending decision older than **14 heartbeats** (≈14 days at daily cadence) is considered stale. The `contrarian`'s loop includes a dedicated scan for aged decisions each heartbeat. For each stale pending decision, the contrarian:

- Proposes supersession if a fresher equivalent exists in the recent history
- Proposes rejection (via `decision-rejection-proposed`) if it's no longer actionable
- Writes a one-line challenge note explaining why it's still relevant if it should stay pending

This prevents the queue from ossifying with decisions the operator will never address but won't explicitly close.

## Shared State
Under `store/teams/monetization/shared/`:

- `TEAM.md` — this file
- `decisions.jsonl` — standard team decision stream
- `knowledge.jsonl` — durable knowledge entries (e.g., assumption updates, benchmark captures)
- `handoff-history.jsonl` — per-run handoffs from each member
- `ledger.jsonl` — financial-tracker's heartbeat snapshots (cost, MRR, time allocation, runway, default-alive gap, deltas, flags)
- `opportunities.jsonl` — opportunity-scout's idea pool with SKU classification and proposed revisit triggers
- `market-scans.jsonl` — market-validator's benchmark captures, competitive observations, assumption validations
- `operator-inputs.json` — operator-provided financial state (cash, burn categories, time allocation, services data). Operator edits directly per the carve-out in operating rule 3. Gathering guidance: [`docs/monetization/HOW_TO_GATHER_INPUTS.md`](../../../../../../docs/monetization/HOW_TO_GATHER_INPUTS.md).

Durable canonical docs live at project level in `docs/monetization/` — read-only for the team during heartbeats, editable only by the operator via accepted decisions.

### Knowledge supersession policy

Members emit snapshot-style knowledge entries every heartbeat. To prevent `knowledge.jsonl` from bloating with daily near-duplicates, snapshot entries **must** reference the prior same-type entry via the `"supersedes"` field, matching the director-swarm pattern.

Topic families that supersede:

- `catalog-snapshot-YYYY-MM-DD` (catalog-strategist) — supersedes the most recent `catalog-snapshot-*`
- `ledger-snapshot-YYYY-MM-DD` (financial-tracker) — supersedes the most recent `ledger-snapshot-*`
- `scout-scan-YYYY-MM-DD` (opportunity-scout) — supersedes the most recent `scout-scan-*`
- `market-scan-YYYY-MM-DD` (market-validator) — supersedes the most recent `market-scan-*`

Topic families that do **not** supersede (append-only historical record):

- `challenge-note/<decision-id>` (contrarian) — one per challenged decision, kept forever
- `decision-application/<decision-id>` — one per applied decision, kept forever
- Any operator-authored knowledge entry — kept forever

Operational exhaust in `.jsonl` files outside `knowledge.jsonl` (ledger.jsonl, opportunities.jsonl, market-scans.jsonl, handoff-history.jsonl, decisions.jsonl) is append-only time-series and never supersedes. Supersession applies only to snapshot-style entries in `knowledge.jsonl`.

## Source-of-truth Docs (canonical)
These are under `docs/monetization/` at the repo root. Paths below are relative to the repo root; members' HEARTBEAT.md files reference them via the `DOCS_ROOT` pointer below.

Relative to repo root:
- [`docs/monetization/STRATEGY.md`](../../../../../../docs/monetization/STRATEGY.md) — narrative + principles + long-term directions
- [`docs/monetization/CATALOG.md`](../../../../../../docs/monetization/CATALOG.md) — SKU index + lifecycle + guardrails
- [`docs/monetization/catalog/base/business.md`](../../../../../../docs/monetization/catalog/base/business.md) — business bundle detail
- [`docs/monetization/catalog/base/lifestyle.md`](../../../../../../docs/monetization/catalog/base/lifestyle.md) — lifestyle bundle detail
- [`docs/monetization/catalog/addons/`](../../../../../../docs/monetization/catalog/addons/) — add-on candidate files
- [`docs/monetization/TIERS.md`](../../../../../../docs/monetization/TIERS.md) — delivery tiers
- [`docs/monetization/PRICING.md`](../../../../../../docs/monetization/PRICING.md) — tier × bundle matrix
- [`docs/monetization/FINANCIAL_MODEL.md`](../../../../../../docs/monetization/FINANCIAL_MODEL.md) — cost structure, runway math, default-alive
- [`docs/monetization/FUNNEL.md`](../../../../../../docs/monetization/FUNNEL.md) — AARRR-adapted stages, owners, metrics
- [`docs/monetization/REVENUE_LINES.md`](../../../../../../docs/monetization/REVENUE_LINES.md) — subscription + services lines with discipline rules
- [`docs/monetization/TELEMETRY_ROADMAP.md`](../../../../../../docs/monetization/TELEMETRY_ROADMAP.md) — metric-to-capability gap map
- [`docs/monetization/BENCHMARKS.md`](../../../../../../docs/monetization/BENCHMARKS.md) — market-validator's curated benchmarks
- [`docs/monetization/HOW_TO_GATHER_INPUTS.md`](../../../../../../docs/monetization/HOW_TO_GATHER_INPUTS.md) — per-field guidance for `shared/operator-inputs.json`
- [`docs/monetization/scenario-sku-map.json`](../../../../../../docs/monetization/scenario-sku-map.json) — scenario-to-SKU many-to-many mapping

**DOCS_ROOT:** `docs/monetization/` (from repo root). Member HEARTBEAT.md files reference this path.

## Cross-Team Coordination
The monetization team is the **canonical source** for monetization state. Other teams consume its outputs:

- **director-swarm** reads `CATALOG.md` for the revenue critical path rather than deriving it ad-hoc. Specifically wired today: `portfolio-manager` (reads CATALOG + business.md + scenario-sku-map.json each heartbeat to weight Now/Near/Far) and `vision-walk-prep` (reads CATALOG for the bundle-roadmap section of the morning briefing). `outcome-strategist` will consume monetization signals once it is re-enabled alongside Command Center — not wired today because the member itself is disabled.
- **marketing-crew** reads `CATALOG.md` + `STRATEGY.md` for positioning.
- **scenario-qa** has no direct dependency; indirectly their quality work affects depth-layer readiness.
- **landing-page-business-suite** (scenario) reads `CATALOG.md` + `PRICING.md` + `TIERS.md` to generate pricing pages and entitlements.
- **scenario-to-cloud** (scenario) reads `TIERS.md` to understand what deployment-mode readiness the monetization plan depends on.

The monetization team does **not** call into other teams. It surfaces decisions that may affect them; the operator routes.

## Heartbeat Coordination Principle
Because this team is leaderless, there is no "every member must wait for the lead before emitting." Each member:

- Emits its own first-class output (ledger entry, opportunity entry, catalog deltas, benchmark update, skeptical review)
- Runs its own decision stream
- Does not aggregate other members' work

The morning vision walk (via `vision-walk-prep`) is the aggregation layer. Individual members should not attempt to pre-synthesize into a "monetization brief" — that is the human's job at the vision walk.

## Key Skills
Read the relevant skill before starting a task. Each skill contains usage instructions and current capabilities.

- `prompt-manager skill read documentation-health` — keep decisions, proposals, and knowledge entries concrete and readable
- `prompt-manager skill read swarm-manager-backlog-tools` — initiative/backlog inspection (catalog-strategist uses this to read portfolio state)
- `prompt-manager skill read systematic-exploration` — when opportunity-scout scans market landscape
- `prompt-manager skill read scientific-debugging` — when financial-tracker or market-validator needs to explain an unexpected delta

## Anti-Patterns to Avoid
- **Synthesizing other members' outputs.** That's the leader-led antipattern this team explicitly avoids.
- **Hallucinating current-state metrics.** Use `pending-telemetry` instead.
- **Promoting candidates without trigger firing.** Guardrail violation.
- **Framing paid subscription as paywalling core features.** Positioning violation.
- **Proposing marketing-driven expansion when an agent surface could do it.** Inefficient; use the structural advantage.
- **Building new telemetry scenarios ahead of need.** Premature infrastructure.
- **Scoping add-ons before parent bundle has paying users.** Focus discipline violation.
