# React Component Library

The React Component Library is Vrooli's shared, versioned UI capability. An
asset is authored once, validated at its source, and consumed by scenario UIs
through a governed package surface.

## The asset loop

1. Declare or update the asset intent in `catalog/assets/<domain>/<asset>.json`.
2. Open a writable draft, edit the implementation and story, and keep any CSS
   companion beside the implementation.
3. Run the generator, validate the changed asset, and publish a new immutable
   version.

```bash
react-component-library components draft-begin react-component-library:Button
react-component-library asset check controls.button
react-component-library components draft-publish react-component-library:Button
```

Only two surfaces are authored: the catalog declaration and the version
directory. `component.json`, dependency locks, story contracts, package
exports, release hashes, provenance, and database projections are generated
by `react-component-library catalog build`. Never edit a released version or a
derived artifact by hand.

## Start the scenario

```bash
make setup
make start
react-component-library status
```

Use [docs/guides/asset-update-flow.md](docs/guides/asset-update-flow.md) for
the complete change procedure and [docs/reference/cli-commands.md](docs/reference/cli-commands.md)
for the command surface. Run the scenario-owned suite with
`vrooli scenario test react-component-library` when a full workflow check is
needed.

## Always and never

- Always edit through `components draft-begin` and publish through the governed
  lifecycle.
- Always run `catalog build --check` before handing off a change.
- Always use the narrowest applicable gate first; a zero-input result is a
  runner failure, not evidence of quality.
- Always treat the catalog as desired intent and source as observed behavior.
- Never copy catalog metadata into `component.json` or a source header.
- Never add a reconciliation script or an exemption to hide drift.
- Never use exact intra-library version pins in newly authored source; use a
  supported major line.
- Never place scratch output in the scenario tree.

## Package boundary

The published package is built from the authored library and emits JavaScript
and declarations. Authored CSS is inlined into the emitted module through the
library stylesheet injector; CSS files are not required from a consumer's
bundler. Package maintenance tooling lives in
`packages/react-component-library/tooling/`.

## Further reading

- [Asset derivation](docs/concepts/ASSET-DERIVATION.md) — ownership and
  generated projections.
- [CLI reference](docs/reference/cli-commands.md) — lifecycle setup and operations.
- [Architecture](docs/concepts/ARCHITECTURE.md) — API, CLI, UI, and storage.
- [Testing](docs/internal/TESTING.md) — targeted and scenario-owned checks.
