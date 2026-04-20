# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **swarm-manager-backlog-tools** — portfolio and initiative inspection
- **documentation-health** — durable, grep-able catalog deltas
- **systematic-exploration** — when assessing whether a scenario has crossed a readiness threshold

## Primary Surfaces
- `docs/monetization/CATALOG.md`
- `docs/monetization/catalog/base/*.md`
- `docs/monetization/catalog/addons/*.md`
- `docs/monetization/TIERS.md`
- `docs/monetization/scenario-sku-map.json`
- `swarm-manager overview`
- `swarm-manager initiatives list`
- `swarm-manager initiatives get --name <name>`
- `prompt-manager team decision-list monetization --status=pending --context=catalog-promotion`
- `prompt-manager team decision-list monetization --status=pending --context=catalog-mapping-update`
- `prompt-manager team knowledge-list monetization`
- `ls scenarios/` + per-scenario `service.json` / `PRD.md`

## Usage Rules
- Do not edit canonical docs under `docs/monetization/`. Propose edits via decisions; operator curates.
- Every numeric / readiness claim carries an honesty flag: `fixed` / `measured` / `estimate` / `pending-telemetry`.
- Do not aggregate other members' outputs.
- Cap decisions at 3 per heartbeat.
- When `scenario-to-cloud` exposes a structured readiness query, migrate qualitative readiness judgments by searching for `REPLACES-MANUAL` in your prompts and updating them.
