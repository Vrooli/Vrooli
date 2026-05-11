# Marketing PoR Adoption Validation

## Validation Commands

```bash
prompt-manager graph operating-model validate --team marketing-crew --id marketing-operating-model
prompt-manager graph operating-model diff --team marketing-crew --id marketing-operating-model
prompt-manager graph operating-model coverage --team marketing-crew --id marketing-operating-model
```

When PoR manifest validation lands, run the marketing PoR validator against [`../manifest.json`](../manifest.json) and the shared base contract at [`../../agent-system/team-plan-of-record.manifest.json`](../../agent-system/team-plan-of-record.manifest.json).

## Expected Clean State

- `README.md` is the only top-level prose entrypoint.
- Durable canon lives under `operating/`, `taxonomies/`, `methods/`, `catalogs/`, `strategy/`, `evidence/`, or `governance/`.
- `notebook/` remains a working notebook and is excluded from PoR authority.
- `taxonomies/marketing-research/README.md` and `taxonomies/marketing-research/taxonomy.json` remain paired.
- `operating/OPERATING_MODEL.md` remains clean under operating-model validation.

## Migration Notes

This folder was migrated from the earlier flat marketing PoR layout. Old top-level strategy files moved under `strategy/`; `post-types/`, `post-techniques/`, `research/`, `rich-media/`, and the signal taxonomy moved into semantic modules declared by [`../manifest.json`](../manifest.json).
