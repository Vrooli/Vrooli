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

## Shell archetypes and maturity

Whole-asset retirement is owned by `components retire <libraryId>`. Run it
without flags for consumer preflight, then with `--apply` to withdraw an unused,
deprecated asset. The operation preserves authored releases, registry-only
history, and the catalog declaration under `library/.retired`; it removes the
active source/catalog/registry entry. JSON output includes archive paths and
any operational error. Package exports require the normal package sync/build
after withdrawal. Test-history reports remain in the registry database.

An interrupted operation leaves a pending record that blocks indexing and
authoring. Repeat the apply command to recover it: restore source/catalog if
registry removal did not commit, or record completion if it did. Ambiguous
state remains blocked with an error and preserved evidence.

Choose a library-owned archetype for the product surface. `AppShell/2` is the navigated-console frame, not a mandate to turn every product into a console. Version numbers describe release history; they do not establish product fit. The reviewed inventory below is dated 2026-09-07.

| Asset | Latest (retained source versions) | Role / declaration | Adoption status |
| --- | --- | --- | --- |
| `AppShell` | 2.0.4 (1.0.7, 2.0.2, 2.0.3, 2.0.4) | navigated-console | **Proven for console migration.** Owns desktop navigation, phone tabs/drawer, utility and routed main pane; use /2. |
| `SidebarShell` | 2.7.2 (1.3.1, 2.7.2) | navigation column | **Composition primitive.** AppShell composes /2. The /1 alias is retired; retained exact releases remain available. It is not a complete application frame. |
| `AmbientDisplayShell` | 0.1.2 (0.1.1, 0.1.2) | ambient-display | **Pre-1.0; unproven migration target.** Do not force adoption. Record a scoped ejection when the board cannot be carried. |
| `AmbientCanvas` | 0.1.1 (0.1.1) | ambient drawing surface | **Pre-1.0 drawing primitive.** Canvas loop, not top-level application chrome. |
| `CommandCenterShell` | 1.0.4 (1.0.4) | command-center | **Available; product fit needs evidence.** Navigation/workspace/status layout. Version 1 alone does not prove fit for an ambient board. |
| `AssetDetailShell` | 1.1.6 (1.1.6) | asset-detail page | **Page composition primitive.** Preview/metadata/actions layout within a host application frame. |
| `MasterDetail` | 1.0.8 (1.0.8) | master-detail | **Available; product fit needs evidence.** Owns list/detail transitions; use as the declared frame only when that is the entire product surface. |
| `InspectorLayout` | 1.1.5 (1.1.4, 1.1.5) | inspector | **Available; product fit needs evidence.** Canvas/toolbar/inspector composition; verify the intended interaction before adoption. |
| `DrawerShell` | 1.1.8 (1.1.8) | drawer | **Overlay composition.** BottomSheet or FullPageDrawer. Appropriate to an embedded drawer experience, not a substitute for console navigation. |
| `CardShell` | 1.0.2 (1.0.0, 1.0.1, 1.0.2) | record card | **Leaf primitive.** Selection and action chrome for a record; not an application shell. |
| `CanvasFrame` | 1.0.2 (1.0.2) | preview canvas | **Preview primitive; inline geometry.** Specimen framing; not an adoptable application archetype. |
| `OverlayCanvas` | 1.0.14 (1.0.14, 1.0.7) | claim visualization | **Leaf visualization.** Subject-bound visualization, not an application frame. |
| `PageFrame` | archived 1.0.3 | preview page frame | **Retired.** Source, catalog declaration, and registry snapshot are preserved under `.retired`; no package export remains. |

`TopBar`, `DashboardPage`, `DetailPage`, and `CollectionPage` are also retired
from the active catalog, registry, and package. `navigation.page` (`Page`) is a
different asset from `PageFrame` and remains available.

For a declared frame, set `shell.archetype`, `shell.asset`, `shell.entry`, and `shell.export` in `ui/manifest.json`. ui-health’s `standard_shell_ownership` rule checks the declared mount and production source. Its current declaration set is `navigated-console`, `ambient-display`, `command-center`, `master-detail`, `inspector`, and `drawer`, mapped to their assets above. The other rows are composition primitives, not additional accepted application archetypes.

When a library archetype cannot preserve the product’s behavior, document a fenced `shell-ejection` JSON block in `docs/reference/component-library-gaps.md`, with `archetype`, a concrete `reason`, and exact scenario-relative `files` under `ui/src/`, `ui/public/`, or supported root UI entry files. The exception applies only to those files and is reported by the rule. Pre-1.0 assets must not be imposed on a working product to satisfy an adoption count.

AppShell migration evidence currently includes targeted library tests, desktop/phone Chromium utility checks, and migrated console source checks. This inventory does not certify the other archetypes or the full fleet migration.

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
