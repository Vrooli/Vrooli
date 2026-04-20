# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **documentation-health** — ledger entries must be durable and machine-readable
- **scientific-debugging** — for unexpected deltas, isolate root cause instead of narrating

## Primary Surfaces
- `docs/monetization/FINANCIAL_MODEL.md`
- `docs/monetization/PRICING.md`
- `docs/monetization/REVENUE_LINES.md`
- `docs/monetization/TELEMETRY_ROADMAP.md`
- `shared/ledger.jsonl` (own history)
- **REPLACES-MANUAL:** `landing-page-business-suite subscriptions summary --format json` when it exists
- **REPLACES-MANUAL:** `scenario-to-cloud costs aggregate --days 30` when it exists
- `prompt-manager team decision-list monetization --status=pending --context=runway-warning`
- `prompt-manager team decision-list monetization --status=pending --context=services-trap-warning`

## Usage Rules
- Every numeric field MUST carry an honesty flag. Unlabeled numbers are guardrail violations.
- Do not invent data. Missing data → `pending-telemetry` with a pointer to the matching `TELEMETRY_ROADMAP.md` gap.
- Cap decisions at 2 per heartbeat.
- When a `REPLACES-MANUAL` capability lands, grep the prompts for the marker and migrate the qualitative step to a structured query.
- Time allocation is required output every heartbeat, same as dollar costs. Do not skip it.
