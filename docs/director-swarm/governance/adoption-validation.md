# Director Swarm PoR Adoption Validation

## Validation Commands

```bash
prompt-manager graph operating-model validate --team director-swarm --id director-swarm-operating-model
prompt-manager graph operating-model diff --team director-swarm --id director-swarm-operating-model
prompt-manager graph operating-model coverage --team director-swarm --id director-swarm-operating-model
```

When PoR manifest validation lands, run the director-swarm PoR validator against [`../manifest.json`](../manifest.json) and the shared base contract at [`../../agent-system/team-plan-of-record.manifest.json`](../../agent-system/team-plan-of-record.manifest.json).

## Expected Clean State

- `README.md` and `manifest.json` are the only top-level files.
- Durable canon lives under `operating/`, `strategy/`, `evidence/`, or `governance/`.
- `strategy/PORTFOLIO_PHILOSOPHY.md` and `strategy/ROADMAP.md` remain strategy canon, not live task status.
- `evidence/OUTCOMES_CHARTER.md` remains outcome framing, not a duplicate Command Center dashboard.
- `operating/OPERATING_MODEL.md` remains clean under operating-model validation.

## Migration Notes

This folder was migrated from the earlier flat director-swarm PoR layout. The portfolio philosophy and roadmap moved under `strategy/`; the outcomes charter moved under `evidence/`; the operating model, manifest, and governance files were added to match the shared team PoR structure.
