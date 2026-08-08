# Marketing Operating

## Operating Documents

| Document | Purpose |
|---|---|
| [`OPERATING_MODEL.md`](OPERATING_MODEL.md) | Team-level operating contract: mission, scope, loops, graph, topic catalog, work and dispositions, external inputs, outputs, feedback loop, gaps, and adoption commands. |

## Validation

Validate the operating model with:

```bash
prompt-manager graph operating-model validate --team marketing-crew --id marketing-operating-model
prompt-manager graph operating-model diff --team marketing-crew --id marketing-operating-model
prompt-manager graph operating-model coverage --team marketing-crew --id marketing-operating-model
```

The plan-of-record manifest at [`../manifest.json`](../manifest.json) declares this folder as the operating-contract module.
