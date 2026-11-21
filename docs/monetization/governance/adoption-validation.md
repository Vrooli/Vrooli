# Monetization Adoption Validation

## Validation Commands

Use these commands after changing this plan of record:

```bash
prompt-manager graph operating-model validate --team monetization --id monetization-operating-model
prompt-manager graph operating-model diff --team monetization --id monetization-operating-model
prompt-manager graph operating-model coverage --team monetization --id monetization-operating-model
```

For a quick local tree check, also run:

```bash
find docs/monetization -maxdepth 3 -type f | sort
prompt-manager graph topics
```

## Expected Clean State

- `README.md` is the only top-level prose canon file.
- `manifest.json` declares every durable module.
- `operating/OPERATING_MODEL.md` exists and is registered in the monetization team plan-of-record.
- Taxonomy JSON sidecars live under `taxonomies/<taxonomy-id>/taxonomy.json`.
- SKU, revenue-line, and channel entries live under `catalogs/`.
- Strategy docs live under `strategy/`.
- Benchmarks, financial assumptions, and telemetry gaps live under `evidence/`.

## Catalog reconciliation gate

Before claiming that the monetization instrument is adopted, run the Offer Desk
reconciliation gate against the declared source tree:

```bash
offer-desk offers catalog-verify \
  --source-path docs/monetization \
  --source-mode operator \
  --json
```

The command is read-only. It reports per-file expected/live counts and fails
when duplicate `(kind, name)` identities, orphan edges, extra live nodes, or
import findings remain. A clean adoption receipt requires `reconciled: true`
and `total_drift: 0`; rerun it after any catalog import or identity merge.

## Enforcement Scope

Validation treats [`../manifest.json`](../manifest.json) as the only structural contract for this plan of record. New durable canon must be registered in the manifest, placed under the most specific standard module, and edited only through accepted monetization decisions.
