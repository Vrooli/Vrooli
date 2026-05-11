# Marketing PoR Adoption Validation

## Validation Commands

```bash
prompt-manager graph operating-model validate --team marketing-crew --id marketing-operating-model
prompt-manager graph operating-model diff --team marketing-crew --id marketing-operating-model
prompt-manager graph operating-model coverage --team marketing-crew --id marketing-operating-model
```

Operating-model validation consumes the local [`../manifest.json`](../manifest.json) and the shared base contract at [`../../agent-system/team-plan-of-record.manifest.json`](../../agent-system/team-plan-of-record.manifest.json).

## Expected Clean State

- `README.md` is the only top-level prose entrypoint.
- Durable canon lives under `operating/`, `taxonomies/`, `methods/`, `catalogs/`, `strategy/`, `evidence/`, or `governance/`.
- Non-canon observations use typed team knowledge topics and enter the PoR only through accepted decisions.
- `taxonomies/marketing-research/README.md` and `taxonomies/marketing-research/taxonomy.json` remain paired.
- `operating/OPERATING_MODEL.md` remains clean under operating-model validation.

## Enforcement Scope

Validation treats [`../manifest.json`](../manifest.json) as the only structural contract for this plan of record. New durable canon must be registered in the manifest, placed under the most specific standard module, and edited only through accepted marketing decisions.
