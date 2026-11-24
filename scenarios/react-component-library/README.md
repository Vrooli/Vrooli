# React Component Library

Browse, author, inspect and compose Vrooli's versioned React assets. The workbench separates visual components, hooks and supporting assets, renders declared stories, and exposes source-backed page verification. Start it with `make start` from this scenario directory.

## Workbench — five destinations and one workspace model

The workbench now has `num[sot]:five` primary destinations: **Start, Library, Design, Review, and Stats**. Start combines search, intent browsing, recent workspaces, curated templates, library pulse, and shortcuts. Coverage, Activity, and Capabilities are views inside Stats; Review remains the queue for findings that need a decision.

Design work is stored as a **Workspace → Page → Display → State** aggregate. Curated templates are immutable workspace snapshots; using one forks an editable workspace. The studio supports composition, import of HTML/CSS regions, RCL component placement and replacement, revision history, state capture, and visual comparison. The current source for the eight-screen design study is retained with its matching styles, scripts, and reference image in the `rcl-eight-screen-workspace-20261121` experience-engineering operator experiment.

This describes implemented behavior, not a blanket production-readiness claim. Authenticated BAS capture and independent review of AI answers remain open evidence requirements; imported source interactions are not executed as native workspace behavior. See the [workbench implementation program](docs/internal/WORKBENCH-PROGRAM.md) for current evidence, migration boundaries, and unresolved work, and the [maturation prototype](docs/mockups/workbench-maturation.html) for the design reference.

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
- [Search and composition proposal](docs/concepts/SEARCH-AND-COMPOSITION.md) and [visual options](docs/mockups/search-options.html)
- [Workbench maturation: five destinations + workspace model](docs/concepts/WORKBENCH-MATURATION.md) and [full prototype](docs/mockups/workbench-maturation.html)
- [Workbench implementation program](docs/internal/WORKBENCH-PROGRAM.md) — current state, target architecture, migration, deletion ledger and quality gates
- [Edit, rebuild and look](docs/guides/asset-update-flow.md#edit-rebuild-and-look)
- [Architecture and owners](docs/concepts/ARCHITECTURE.md)
- [Tokens](docs/reference/token-contract.md) and [stamps/capture](docs/reference/cli-commands.md#inspect-a-running-page)
- [Page/source reconciliation](docs/concepts/UI-SPEC-RECONCILIATION.md)
- [Story contract](docs/concepts/STORY-CONTRACT.md)
- [Known problems and evidence limits](docs/internal/PROBLEMS.md)
- [Studio debt inventory](docs/studio/DEBT.md) and the [Nov 19 historical audit](docs/studio/DEBT-HISTORY-2026-11-19.md). Studio handoffs in `docs/studio/` are retained as historical context; current operating guidance lives in the workbench program and problems ledger.
- [Design intent](DESIGN.md)
