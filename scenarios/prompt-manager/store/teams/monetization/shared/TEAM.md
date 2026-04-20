# Monetization Team

## Mission
Own the canonical monetization plan for Vrooli — catalog, tiers, funnel, revenue lines, and financial model — and surface the decisions the operator must make each heartbeat to stay on a path to default-alive. The operator is the real strategist; this team maintains the plan, tracks current state against it, and converts measurable changes into concrete decisions.

## Coordination Pattern
Leaderless / independent. Five members, each with its own heartbeat and its own decision stream. There is no AI lead — do not recreate one implicitly through "synthesize the other agents" behavior. Coordination happens outside the team, in the morning vision walk, where the operator reviews pending decisions across all members.

If a member is tempted to aggregate other members' outputs into a single brief, that is the leader-led antipattern. Each member stays in its own lane and produces its own first-class output.

## Members
- **catalog-strategist** — maintains the SKU / tier / scenario graph and proposes promotions when triggers fire.
- **opportunity-scout** — generates and classifies candidate SKUs, add-ons, and services lines.
- **financial-tracker** — maintains the ledger (costs, revenue, time allocation, runway, default-alive gap).
- **market-validator** — grounds pricing and retention assumptions in external benchmarks for the active tier.
- **contrarian** — mandatory skeptic across all other members' outputs and pending decisions.

Each member has an AGENTS.md, SOUL.md, TOOLS.md under `store/agents/<member>/` and a RESPONSIBILITIES.md + HEARTBEAT.md under `store/teams/monetization/members/<member>/`.

## Operating Rules

1. **Default focus is `active` SKUs and the active tier.** Candidate SKUs and candidate tiers are read only to check their revisit triggers. Do not re-evaluate dormant candidates every heartbeat.
2. **No candidate without an explicit revisit trigger.** When opportunity-scout captures a candidate or catalog-strategist promotes an idea to candidate, a concrete trigger condition must be attached. Vibes are not triggers.
3. **Agents propose doc edits via decisions. The operator curates canonical docs.** No member writes directly to the files in `docs/monetization/`. Doc changes happen through decisions with contexts listed below.
4. **Honesty flags are mandatory on every number.** Every metric emitted is labeled `fixed`, `estimate`, `aspirational`, `measured`, or `pending-telemetry`. Unlabeled numbers are guardrail violations.
5. **Open-source self-host is strategic positioning.** Frame the subscription as convenience + integrated gateway. Do NOT frame it as paywalling core features. If an output reads as "users who go free are leaking revenue," the framing is broken.
6. **Agents are the expansion engine.** When proposing acquisition, activation, upsell, or retention mechanisms, default to agent-driven surfaces. Fall back to marketing/email/pop-ups only when agents cannot reach the relevant moment.
7. **Services are a bridge, not a business.** Every active services revenue line must have a hypothesis, a fixed-duration pilot, a productization target, and a sunset/convert clause. Services revenue is tracked separately from subscription revenue.
8. **Tier 4 (hardware) is north-star.** Do not plan work against it without explicit operator initiation.
9. **Every scenario proposal articulates both acquisition AND retention impact.** Scenarios evaluated only on acquisition appeal will starve the retention side of the funnel.
10. **Activation work is retention work.** Most churn is failed activation. When the team proposes retention investments, check whether the real issue is activation first.
11. **Downgrade-to-free is not churn.** Track separately. Different signal, different causes.
12. **Legal surface check on every services line.** Lead-gen (TCPA/CAN-SPAM/GDPR), consulting (contract/IP), done-for-you builds (warranty/liability) all carry distinct regulatory exposure.
13. **Services capacity ≤ 30% of time budget.** Exceeding for 3+ consecutive weeks triggers a services-trap review.
14. **Pre-launch metrics are aspirational.** Do not hallucinate current-state numbers for unmeasured metrics. Use `pending-telemetry` and point at [`TELEMETRY_ROADMAP.md`](../../../../../../docs/monetization/TELEMETRY_ROADMAP.md) instead.
15. **This team does not build telemetry scenarios.** Capability gaps are captured in TELEMETRY_ROADMAP.md; scenarios are built (or not) by feature/QA/portfolio lanes when their priority justifies it.

## Decision Contexts
Members surface decisions with these contexts. The operator reviews them at the morning vision walk.

- `catalog-promotion` — scenario gains headliner status, candidate SKU or tier gets trigger-met and is proposed for promotion
- `catalog-mapping-update` — scenario-sku-map.json change proposal
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

Keep decision descriptions short, concrete, and tied to a specific action the operator can take or defer.

## Shared State
Under `store/teams/monetization/shared/`:

- `TEAM.md` — this file
- `decisions.jsonl` — standard team decision stream
- `knowledge.jsonl` — durable knowledge entries (e.g., assumption updates, benchmark captures)
- `handoff-history.jsonl` — per-run handoffs from each member
- `ledger.jsonl` — financial-tracker's heartbeat snapshots (cost, MRR, time allocation, runway, default-alive gap, deltas, flags)
- `opportunities.jsonl` — opportunity-scout's idea pool with SKU classification and proposed revisit triggers
- `market-scans.jsonl` — market-validator's benchmark captures, competitive observations, assumption validations

Durable canonical docs live at project level in `docs/monetization/` — read-only for the team during heartbeats, editable only by the operator via accepted decisions.

## Source-of-truth Docs (canonical)
These are under `docs/monetization/` at the repo root. Paths below are relative to the repo root; members' HEARTBEAT.md files reference them via the `DOCS_ROOT` pointer below.

Relative to repo root:
- [`docs/monetization/STRATEGY.md`](../../../../../../docs/monetization/STRATEGY.md) — narrative + principles + north-stars
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
- [`docs/monetization/scenario-sku-map.json`](../../../../../../docs/monetization/scenario-sku-map.json) — scenario-to-SKU many-to-many mapping

**DOCS_ROOT:** `docs/monetization/` (from repo root). Member HEARTBEAT.md files reference this path.

## Cross-Team Coordination
The monetization team is the **canonical source** for monetization state. Other teams consume its outputs:

- **director-swarm** reads `CATALOG.md` for the revenue critical path rather than deriving it ad-hoc each heartbeat.
- **scenario-feature** reads `CATALOG.md` and `scenario-sku-map.json` before scoping new work, so features map to bundle impact.
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
