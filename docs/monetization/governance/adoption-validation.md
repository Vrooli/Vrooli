# Monetization Adoption Validation

## Validation Commands

Use these commands once prompt-manager PoR validation consumes local manifests:

```bash
prompt-manager graph operating-model validate --team monetization --id monetization-operating-model
prompt-manager graph operating-model diff --team monetization --id monetization-operating-model
prompt-manager graph operating-model coverage --team monetization --id monetization-operating-model
```

Until the PoR validator lands, manually verify:

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

## Migration Notes

This folder was migrated from the older flat layout where strategy, catalog, taxonomies, evidence, and governance files all lived at the top level. Historical handoff logs may still cite old paths; update active prompts, team config, and docs references first.
