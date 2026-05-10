# Temporal Model Artifacts

This directory owns the template-local formal temporal-model validation loop.

`*.flow.json` files are the source of truth. Generated `.qnt` files and
`*.formal.generated.json` artifacts are checked-in review artifacts and must not
be edited by hand.

Run:

```bash
node tools/temporal-model/generate.mjs --list
node tools/temporal-model/generate.mjs --flow notes.attachment-upload.ui
node tools/temporal-model/generate.mjs
node tools/temporal-model/generate.mjs --check
```

The generator runs `quint typecheck`, `quint test`, `quint verify`, and
`quint run --mbt` for each discovered temporal flow contract. It renders Quint
from the contract, normalizes ITF traces, records source/model/generator hashes,
and writes checked-in `*.formal.generated.json` artifacts beside the workflow
they validate.

To update a flow:

1. Edit the domain-local `*.flow.json` contract.
2. Update the production transition function.
3. Run `node tools/temporal-model/generate.mjs --flow <flow-id>`.
4. Run `node tools/temporal-model/generate.mjs --check`.
5. Run the scenario tests on an instantiated scenario.
