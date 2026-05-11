# Infra Health Adoption Validation

## Validation Commands

Use these commands after changing this plan of record:

```bash
prompt-manager graph operating-model validate --team infra-health --id infra-health-operating-model
prompt-manager graph operating-model diff --team infra-health --id infra-health-operating-model
prompt-manager graph operating-model coverage --team infra-health --id infra-health-operating-model
```

For a quick local tree check, also run:

```bash
find docs/infra-health -maxdepth 3 -type f | sort
prompt-manager graph topics
```

## Expected Clean State

- `README.md` is the only top-level prose canon file.
- `manifest.json` declares every durable module.
- `operating/OPERATING_MODEL.md` exists and is registered in the infra-health team plan-of-record.
- Reliability targets live under `strategy/`.
- Instrumentation gaps and cross-platform debt live under `evidence/`.
- Editing authority, validation notes, and migration history live under `governance/`.

## Enforcement Scope

Validation treats [`../manifest.json`](../manifest.json) as the only structural contract for this plan of record. New durable canon must be registered in the manifest, placed under the most specific standard module, and edited only through accepted infra-health decisions.
