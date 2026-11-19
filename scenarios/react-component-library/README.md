# React Component Library

Browse, author, inspect and compose Vrooli's versioned React assets. The workbench separates visual components, hooks and supporting assets, renders declared stories, and exposes source-backed page verification. Start it with `make start` from this scenario directory.

## Workbench redesign — D · Nocturne (implemented and served; governed capture pending)

The operator selected **D · Nocturne** as the target workbench design on 2026-11-18. The target is a warm, dark editorial workbench with eleven pages: Home, Components, Component, Design, Review, Coverage, Capabilities, Activity, Settings, Create and Preview. It keeps the component's two working modes — **Fixtures** and **In context** — and moves the seven current asset tabs into an **Asset data** drawer.

The Nocturne shell, eleven page surfaces, component workspace, design studio and eight page templates are implemented in [`ui/src/workbench/`](ui/src/workbench/). The Design studio can compose, inspect, edit, duplicate and present those templates, and In context places the real component harness inside them where it can be dragged, resized and reset. Fixtures, layers and browser-local templates persist across a reload. The coverage report now persists its last verdict and reports an explicit “preparing” state instead of failing or rendering zero on a cold pass.
This is **not yet a production-ready claim**. The governed capture service returns `503` on this host, so page/component pixel and accessibility evidence is not fresh; full phone journeys, durable activity attribution and the Create write path remain open. The full page map, component workspace, design system, data contracts, migration phases and feature-preservation ledger are in [docs/concepts/WORKBENCH-REDESIGN.md](docs/concepts/WORKBENCH-REDESIGN.md). The selected interactive prototype and the page-by-page decision review are in the experience-engineering operator experiment:

```text
~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-workbench-redesign-20261118/
```

Current behavior remains what the checked behavior register proves, not what the redesign promises.

## The asset loop

Use the [asset update flow](docs/guides/asset-update-flow.md): open a governed draft, edit through `content-set`, validate the affected story/gate, publish a successor, compile the package, build the UI, and inspect the rendered page. The package build and UI build are separate steps.

```bash
react-component-library components draft-begin react-component-library:Button --json
react-component-library page inspect react-component-library / --json
react-component-library components test page:CoveragePage --version workspace --json
```

`page inspect` joins captured stamps to consumer-resolved library source. Page stories use the same declarative evaluator as asset stories, with explicit API fixtures installed before the app mounts. Read the [checked behavior register](docs/internal/TESTING.md#behavior-claim-register) for the enforcing checks and their limits.

## Always and never

Use the catalog for desired capability, manifests for library identity and release pointers, and source evidence for observed implementation. Shared visual values are authored in BaseStyles; generated token files are checked against it. Application-only geometry is authored separately.

Use governed draft/publication and dependency operations. Never rewrite an immutable release or its hashes to make a check pass. Unknown provenance, zero source coverage and unavailable capture/report producers must remain visible findings.

These rules are enforced at the boundaries named in the checked behavior register. Broader visual quality, product fit, production readiness and corpus-wide maturity remain design intent until their relevant checks and review evidence pass.

## Further reading

- [Workbench redesign plan](docs/concepts/WORKBENCH-REDESIGN.md)
- [Edit, rebuild and look](docs/guides/asset-update-flow.md#edit-rebuild-and-look)
- [Architecture and owners](docs/concepts/ARCHITECTURE.md)
- [Tokens](docs/reference/token-contract.md) and [stamps/capture](docs/reference/cli-commands.md#inspect-a-running-page)
- [Page/source reconciliation](docs/concepts/UI-SPEC-RECONCILIATION.md)
- [Story contract](docs/concepts/STORY-CONTRACT.md)
- [Known problems and evidence limits](docs/internal/PROBLEMS.md)
- [Design intent](DESIGN.md)
