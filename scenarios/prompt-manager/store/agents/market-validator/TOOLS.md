# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **systematic-exploration** — broad competitor / category scans
- **documentation-health** — durable benchmark entries
- **scientific-debugging** — when captured benchmark conflicts with an assumption

## Primary Surfaces
- `docs/monetization/BENCHMARKS.md`
- `docs/monetization/PRICING.md`
- `docs/monetization/FINANCIAL_MODEL.md`
- `docs/monetization/FUNNEL.md`
- `docs/monetization/CHANNELS.md`
- `docs/monetization/channels/*.md`
- `shared/market-scans.jsonl`
- Competitor pricing pages (browser fetch)
- Public SaaS benchmark reports (Bessemer State of the Cloud, OpenView PLG benchmarks, SaaS Capital, etc.)
- **REPLACES-MANUAL:** an automated competitor-pricing watcher if one ever exists — see `TELEMETRY_ROADMAP.md` Gap 8

## Usage Rules
- Every captured value has a **source** and a **date**.
- **Applicability** flag is honest: how directly does this comp to Vrooli (high / medium / low)?
- Conflicting data reported as conflicting, not averaged.
- Deep research scoped to active tier × bundle. Everything else is a one-liner.
- Channel research stays scoped to active channels or candidate channels whose trigger is close to firing; otherwise capture a one-line note.
- Cap decisions at 2 per heartbeat.
- Do not build feature-by-feature comparison sheets unless specifically requested.
