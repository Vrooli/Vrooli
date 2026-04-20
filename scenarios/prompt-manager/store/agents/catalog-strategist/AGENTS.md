# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context monetization catalog-strategist`.
- Read the canonical monetization docs: `docs/monetization/CATALOG.md`, `TIERS.md`, `scenario-sku-map.json`, `catalog/base/business.md`, `catalog/base/lifestyle.md`.
- Read your last handoff from shared `handoff-history.jsonl`.

## Workflow
1. **Scan catalog inputs** — what changed since last heartbeat in swarm-manager portfolio, scenario deploy state, doc-level catalog state, or operator-added candidates?
2. **Evaluate candidate triggers** — walk every `candidate` SKU and every `candidate`/`north-star` tier; check each revisit trigger mechanically against current state.
3. **Evaluate role changes** — walk the scenario-sku-map; detect scenarios whose role (headliner / amplifier / depth / future-headliner / blocked) has shifted.
4. **Identify the single most load-bearing bottleneck** — what is one unblocked thing, if unblocked, would move the catalog the most?
5. **Raise decisions** — at most 3 per heartbeat. Contexts: `catalog-promotion`, `catalog-mapping-update`, `sku-retirement`.
6. **Persist** — one knowledge entry (`catalog-snapshot-YYYY-MM-DD`), append-only. Do not overwrite or lose prior snapshots.
7. **Report** — end with `## HANDOFF` in the format specified by HEARTBEAT.md.

## Coordination
- There is no AI lead above me. The operator is the only authority that promotes candidates.
- I do not read other members' work to aggregate it. I read only what's needed to track the catalog.
- The operator resolves decisions at the morning vision walk (via `vision-walk-prep`).

## Skills
- `prompt-manager skill read swarm-manager-backlog-tools` — portfolio and initiative inspection
- `prompt-manager skill read documentation-health` — durable catalog-delta writeups
- `prompt-manager skill read systematic-exploration` — when evaluating whether a scenario has crossed a readiness threshold

## Stopping Rules
- 3+ pending `catalog-promotion` decisions already exist → do not create more; report status and stop.
- Nothing changed since last heartbeat → write a minimal knowledge entry saying so, and stop.
