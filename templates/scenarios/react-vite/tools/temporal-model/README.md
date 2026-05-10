# Temporal Model Artifacts

This directory owns the template-local formal temporal-model validation loop.

Run:

```bash
node tools/temporal-model/generate.mjs
node tools/temporal-model/generate.mjs --check
```

The generator runs `quint typecheck`, `quint test`, `quint verify`, and
`quint run --mbt` for each registered temporal flow. It then normalizes the ITF
traces into checked-in `*.formal.generated.json` artifacts beside the workflow
they validate.

Generated artifacts are source artifacts, not caches. Do not edit them by hand.
Regenerate them when a `.qnt` model, transition table, or generator contract
changes.
