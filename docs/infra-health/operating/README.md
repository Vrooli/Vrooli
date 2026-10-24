# Infra Health Operating

## Operating Documents

| Document | Purpose |
|---|---|
| [`OPERATING_MODEL.md`](OPERATING_MODEL.md) | Team-level operating contract: mission, scope, loops, graph, topic catalog, work and dispositions, inputs, outputs, feedback loop, gaps, and adoption commands. |

## Validation

Validate the operating model with:

```bash
prompt-manager graph operating-model validate --team infra-health --id infra-health-operating-model
prompt-manager graph operating-model diff --team infra-health --id infra-health-operating-model
prompt-manager graph operating-model coverage --team infra-health --id infra-health-operating-model
```

The plan-of-record manifest at [`../manifest.json`](../manifest.json) declares this folder as the operating-contract module.
