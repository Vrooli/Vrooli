# Meta-Optimization Operating Contract

This folder contains the enforceable operating contract for the `meta-optimization` team.

## Operating Documents

| Document | Purpose |
|---|---|
| [`OPERATING_MODEL.md`](OPERATING_MODEL.md) | Team-level operating graph, topic catalog, work catalog, external inputs, downstream outputs, feedback loop, implementation gaps, and adoption commands. |

## Validation

Run the operating-model checks from the repository root after changing this folder:

```bash
prompt-manager graph operating-model validate --team meta-optimization --id meta-optimization-operating-model
prompt-manager graph operating-model coverage --team meta-optimization --id meta-optimization-operating-model
```
