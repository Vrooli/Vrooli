# UI selectors

`@vrooli/ui-selectors` owns the framework-independent selector registry used by
scenario UIs and templates. `packages/api-core/uiselectors` consumes its portable
manifest in Go. Both implementations run `conformance.json` as regression evidence.

```ts
import { createSelectorRegistry, defineDynamicSelector, resolveSelector } from '@vrooli/ui-selectors';
import { librarySelectors } from './selectors.library';
const registry = createSelectorRegistry(
  { editor: { save: 'editor-save' } },
  { editor: { row: defineDynamicSelector({
    description: 'An editor row', testIdPattern: 'row-${id}',
    params: { id: { type: 'string' } },
  }) } },
  librarySelectors,
);
export const selectors = registry.selectors;
export const selectorsManifest = registry.manifest;
// JSX: data-testid={selectors.editor.row({ id })}
// Browser locator: resolveSelector(selectorsManifest, 'editor.row', { id })
```

Literal selectors and `testIdPattern` functions return actual DOM test IDs.
`selectorPattern` functions return CSS. `resolveSelector` always returns CSS.
Parameters are required, typed, finite for numbers, enum-checked, and escaped as
CSS data. Extra parameters, duplicate flattened keys, and conflicting definitions
are errors. A `selectorPattern` placeholder is a CSS value, not a raw CSS fragment;
keep operators and structural syntax in the pattern. Use `scopeSelector(parent,
child)` to keep comma-separated branches within a parent scope.

Library catalog names are namespaces for keys, not prefixes added to DOM IDs.
The third argument composes `library.*` into the same registry and manifest;
application definitions under `library` survive when their keys are disjoint.
A component can declare an explicit `selectors.json` map of semantic names to
actual IDs beside its published version source. React Component Library linking
prefers that contract, otherwise discovers literal IDs in implementation source.
Dynamic IDs and caller-provided IDs need an explicit application contract; source
scanning cannot infer their runtime values. Published component releases remain
immutable.

## Export and discovery

Each UI calls `exportSelectorManifest` from `@vrooli/ui-selectors/export` in its
`selector:manifest` script. `selector:check` uses `{ check: true }` and fails if
checked-in output differs. UI builds regenerate the manifest. The exporter uses
the consumer's TypeScript compiler; it does not bundle or execute an application.
Registry modules must be side-effect-free and their relative imports must be
local TS/JS modules. Source hashes record the registry and imported definitions
so UI Health can detect a stale library composition.

File locations follow the existing UI manifest contract: local `ui/manifest.json`,
otherwise the scenario's declared template UI manifest, then the scenario's
`.vrooli/ui-manifest.json` file overrides. Set `files.selectorRegistry`,
`files.librarySelectors`, and `files.appEntry` there for relocated UIs. Output is
adjacent to the registry as `selectors.manifest.json`. Compatibility discovery
checks `ui/src/constants` and `ui/src/consts` only within the explicit target.
Go callers use `api-core/uimanifest.LoadAt`; no fallback selects another scenario.

BAS references use `@selector/editor.row(id="${@params/id}")`. Compilation freezes
the referenced definition; execution substitutes the raw parameter, validates it,
and then escapes it. References inside evaluate expressions must occupy a complete
quoted JavaScript string. BAS recording converts only unambiguous literal IDs on
the declared application origin; external sites and frames retain their locators.

## Maintenance

Run `vrooli package test ui-selectors`, the TypeScript contract test, and focused
`go test ./uiselectors ./uimanifest` in api-core after shared changes. Migrate a
consumer through Scenario Dependency Analyzer, never a raw package-manager add.
Template dependency declarations use its `template/<id>` surface: the gateway
regenerates the lockfile in a temporary scenario layout, preserves generated
scenario file references, and leaves installation to scenario setup.

## Migration evidence (2026-09-07)

The consolidation covers 89 scenario registries and both React/Vite templates.
Application maps remain local. It removes 24,634 lines from those registry files,
two copied type/helper modules, and 65 identical generic test files. The new
shared contract has 20 Node tests; Go runs the same 14 manifest conformance cases
plus reference/runtime regressions. A Chromium smoke matched seven IDs containing
punctuation, whitespace, newlines, and Unicode through CSS and JavaScript strings.
All 89 scenario manifests passed generation and `selector:check`.

TypeScript checks covered the registries and their consumers. A comparison with
saved pre-migration sources exposed two extra diagnostics: an obsolete Web Console
cast and a missing iOS target identity selector. Both were repaired and rechecked.
Existing diagnostics outside selectors remain; this is not a clean repository-wide
typecheck claim. Concurrent Switchboard map edits were preserved.

Focused api-core, BAS compiler/executor/validator/handler, library adoption, UI Health
composition, dependency gateway, and React/Vite template-generation checks passed.
Broader Test Genie runs remain non-green: BAS `20260907-213021-8ea867d2` reports UI
discovery and its coverage floor; RCL `20260907-213023-23e603c2` reports catalog/test
failures and an API timeout; UI Health `20260907-213022-122303cc` reports discovery,
rule-documentation and provider-wrapper failures; Template Manager
`20260907-213240-57f1e0c5` could not acquire its unit provider. SDA
`20260907-213240-942054bc` returned PASS with advisory findings, not a clean maturity
certificate. Package run `20260907-212851-095f7136` hit unavailable providers and an
unsupported package dependency target; its direct contract tests passed.

Library bridge repairs only change values supported by published source evidence.
Existing IDs with uncertain provenance remain for review. Literal attributes,
common default `testId` props, and explicit versioned contracts are supported;
arbitrary expressions and consumer overrides require an explicit application
contract. A missing shared dependency or unsupported registry composition makes
library linking fail with an actionable instruction instead of writing a broken
import.
