# Temporal Model Artifacts

This directory owns the template-local formal temporal-model validation loop.

`*.flow.json` files are the source of truth. Generated `.qnt` files and
`*.formal.generated.json` artifacts are checked-in review artifacts and must not
be edited by hand.

Run:

```bash
cd tools/temporal-model && GOWORK=off go run . list --root ../..
cd tools/temporal-model && GOWORK=off go run . validate --root ../..
cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow notes.attachment-upload.ui
cd tools/temporal-model && GOWORK=off go run . generate --root ../..
cd tools/temporal-model && GOWORK=off go run . check --root ../..
```

The generator runs `quint typecheck`, `quint test`, `quint verify`, and
`quint run --mbt` for each discovered temporal flow contract. It renders Quint
from the contract, normalizes ITF traces, records source/model/generator hashes,
and writes checked-in `*.formal.generated.json` artifacts beside the workflow
they validate.

Normal unit tests do not require Quint or Java:

```bash
cd tools/temporal-model && GOWORK=off go test ./...
```

To update a flow:

1. Edit the domain-local `*.flow.json` contract.
2. Update the production transition function.
3. Run `cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow <flow-id>`.
4. Run `make temporal-models`.
5. Run the scenario tests on an instantiated scenario.
