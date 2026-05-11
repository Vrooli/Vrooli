# Director Swarm PoR Adoption Validation

## Validation Commands

```bash
prompt-manager graph operating-model validate --team director-swarm --id director-swarm-operating-model
prompt-manager graph operating-model diff --team director-swarm --id director-swarm-operating-model
prompt-manager graph operating-model coverage --team director-swarm --id director-swarm-operating-model
```

Operating-model validation consumes the local [`../manifest.json`](../manifest.json) and the shared base contract at [`../../agent-system/team-plan-of-record.manifest.json`](../../agent-system/team-plan-of-record.manifest.json).

## Expected Clean State

- `README.md` and `manifest.json` are the only top-level files.
- Durable canon lives under `operating/`, `strategy/`, `evidence/`, or `governance/`.
- `strategy/PORTFOLIO_PHILOSOPHY.md` and `strategy/ROADMAP.md` remain strategy canon, not live task status.
- `evidence/OUTCOMES_CHARTER.md` remains outcome framing, not a duplicate Command Center dashboard.
- `operating/OPERATING_MODEL.md` remains clean under operating-model validation.

## Enforcement Scope

Validation treats [`../manifest.json`](../manifest.json) as the only structural contract for this plan of record. New durable canon must be registered in the manifest, placed under the most specific standard module, and edited only through accepted director-swarm decisions.
