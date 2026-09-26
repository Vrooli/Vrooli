# React Component Library

Browse, author and inspect Vrooli's versioned React assets. The workbench separates visual components, hooks and supporting assets, renders declared stories, and exposes source-backed page verification. Start it with `make start` from this scenario directory.

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

- [Edit, rebuild and look](docs/guides/asset-update-flow.md#edit-rebuild-and-look)
- [Architecture and owners](docs/concepts/ARCHITECTURE.md)
- [Tokens](docs/reference/token-contract.md) and [stamps/capture](docs/reference/cli-commands.md#inspect-a-running-page)
- [Page/source reconciliation](docs/concepts/UI-SPEC-RECONCILIATION.md)
- [Story contract](docs/concepts/STORY-CONTRACT.md)
- [Known problems and evidence limits](docs/internal/PROBLEMS.md)
- [Design intent](DESIGN.md)
