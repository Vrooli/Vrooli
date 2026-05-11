# Infra Health Adoption Validation

## Validation Commands

Use these commands once prompt-manager PoR validation consumes local manifests:

```bash
prompt-manager graph operating-model validate --team infra-health --id infra-health-operating-model
prompt-manager graph operating-model diff --team infra-health --id infra-health-operating-model
prompt-manager graph operating-model coverage --team infra-health --id infra-health-operating-model
```

Until the PoR validator lands, manually verify:

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

## Migration Notes

This folder was migrated from the older flat layout where reliability targets, instrumentation roadmap, and cross-platform ledger files all lived at the top level. Historical plan docs may still cite old paths; update active prompts, team config, and durable cross-references first.
