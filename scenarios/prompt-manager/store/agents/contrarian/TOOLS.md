# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **scientific-debugging** — isolate the specific flaw in a proposal rather than produce vague pushback
- **documentation-health** — challenge notes must be concrete and durable

## Primary Surfaces
- `docs/monetization/STRATEGY.md` (principles to check positioning-drift)
- `docs/monetization/FINANCIAL_MODEL.md` (guardrails: 30% services, default-alive buffer, assumptions)
- `docs/monetization/REVENUE_LINES.md` (services discipline)
- `docs/monetization/CHANNELS.md` (discovery-channel discipline)
- `docs/monetization/channels/*.md` (channel-specific triggers, telemetry, anti-patterns)
- `docs/monetization/CATALOG.md` (trigger discipline, candidate rules)
- `docs/monetization/TIERS.md` (tier-activation prereqs)
- `prompt-manager team decision-list monetization --status=pending --json`
- `shared/opportunities.jsonl`, `shared/ledger.jsonl`, `shared/market-scans.jsonl` tails
- `shared/knowledge.jsonl` (own prior challenge notes)

## Analytical Approaches (scoped to the seven failure modes plus channel-activation guardrail)
- **Inversion:** could this proposal trip failure mode X if I read it uncharitably? If no — clear. If yes — specify.
- **Pre-mortem:** if this proposal failed in 6 months, which of the seven failure modes or channel guardrail was the cause?
- **Steelman the proposal first, then attack.** A weak steelman means the challenge won't land.
- **Second-order effects:** especially for services lines — what does success *look like* 3 months in? Does that success exhaust operator time?
- **Channel/revenue separation:** especially for skill registries and OSS discovery — is this really a discovery path feeding subscription, or is the proposal pretending channel exposure is revenue?

## Usage Rules
- Critique proposals, not members. Skepticism is structural, not personal.
- Always name the **specific failure mode**, **specific missing element**, and **specific revision** that would pass.
- For skill-registry proposals, challenge missing standalone installability, missing security/scanner evidence, missing telemetry, or premature claims of subscription lift.
- Do not manufacture objections. Clean proposals get cleared.
- Cap rejection recommendations at 2 per heartbeat.
- Do not invent new failure modes on the fly — use `framework-update` decisions to propose framework evolution instead.
