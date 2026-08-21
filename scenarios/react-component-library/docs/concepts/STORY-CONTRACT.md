# Story Contract

`story.json` is the single, versioned preview and test contract for one
catalog asset version. It replaces `examples.json`, per-example controls, and
inert setup metadata. A story is a valid instance of an asset's schema; it
never declares a competing schema.

## Ownership

| Concern | Owner | May a story change it? |
|---|---|---|
| Public input names, types, defaults, validation, labels, and form layout | `args` schema | No |
| Named visual or behavioural baseline | `stories[].args` | Yes, with valid values only |
| Provider or adapter dependency | `environment` schema and fixture registry | A story may select a declared fixture |
| Internal component state | Public controlled/default API or interaction sequence | No direct mutation |
| Hook state | Hook fixture, action, settle, and observable output | No direct mutation |
| Render assertions | `expect` vocabulary | Yes, with safe assertions only |
| Composition context | `frame` block and catalog fixture | Yes, by naming an existing frame, region, and fixture |

This boundary deliberately prevents both per-story controls and arbitrary
hook/setup execution. The contract is declarative data, not an escape hatch
for running code in the preview iframe.

## Grammar

Every asset version contains exactly one `story.json`. Schema version 1 remains
fully supported. Schema version 2 adds optional story captions and a constrained
code harness seam. Schema version 3 adds declarative composition frames;
versions 1 and 2 continue to parse unchanged.

```json
{
  "schemaVersion": 3,
  "kind": "component",
  "title": "Status Badge",
  "args": {
    "fields": [
      {
        "path": "tone",
        "label": "Tone",
        "kind": "enum",
        "required": true,
        "default": "success",
        "options": ["success", "warning", "info", "danger"]
      },
      {
        "path": "children",
        "label": "Label",
        "kind": "text",
        "required": true,
        "default": { "$text": "Current" },
        "format": "renderable-text"
      }
    ]
  },
  "environment": { "fixtures": [] },
  "frame": {
    "asset": "navigation.page",
    "region": "navigation",
    "fixture": "fixtures.user-directory"
  },
  "stories": [
    {
      "id": "success",
      "name": "Success",
      "description": "The standard positive status treatment.",
      "args": { "tone": "success", "children": { "$text": "Current" } },
      "expect": [{ "kind": "text", "value": "Current" }]
    }
  ]
}
```

### Schema version 3: preview frames

`frame` may appear at the file level or on an individual story. A story-level
frame replaces the file-level declaration. `asset` names a catalog asset that
targets `react-vite`, `region` names one of that asset's declared regions, and
`fixture` names a catalog asset of kind `fixture`. The indexer rejects unknown
assets, undeclared regions, non-fixtures, and fixtures that do not satisfy the
frame's `data-source` type arguments with named diagnostics.

The preview bundles the frame and subject separately into the same isolated
document. The subject is passed in the selected region; the remaining frame
regions receive the declared fixture context. A frame is preview composition
only and is never added to the subject's adoption or dependency closure. The
reference specimen is `navigation.sidebar` framed by `navigation.page`, so a
sidebar is judged as it appears in a real page document rather than as an
orphaned panel.

### Schema version 2: captions and custom harnesses

`stories[].description` is optional explanatory copy displayed below the
specimen title and in the story picker. Omit it rather than rendering an empty
caption.

`stories[].harness` optionally names a JavaScript named export from the one
version-local `story.tsx` file. Direct exports and named re-exports are both
accepted; the story indexer validates the exported name without requiring
preview authors to duplicate a shared harness. The export receives `{ args, log }`, where
`args` is the fully resolved story props (including workbench knob overrides)
and `log(name, ...args)` records an event in the workbench. A harness changes
presentation only: `story.json` remains the sole source of story ids, public
argument schema, interactions, and expectations. Harness files are preview
artifacts; catalog ingestion, adoption closures, and source-parity inventories
exclude them, so demo code never ships to adopters.

```tsx
export function ControlledWithReadout({ args, log }: StoryHarnessProps) {
  const [value, setValue] = useState(args.value)
  return <><ColorPicker {...args} value={value} onChange={(next) => { setValue(next); log("change", next) }} /><output>{value}</output></>
}
```

Harness names must be valid non-reserved JavaScript identifiers. Unknown JSON
fields still fail parsing, including misspelled `description` or `harness`.

For stories without public arguments, omit `args` entirely or write `args: {}`;
the indexer normalizes both forms to the same stored contract. `args: null` is
also normalized to an empty object. This keeps the contract strict about value
types while removing a meaningless schema tax from static stories.

`kind` is either `component` or `hook`. `schemaVersion` is mandatory and is
the compatibility boundary for the parser. Unknown top-level or field keys are
errors; a contract cannot silently acquire meaning from a misspelling.

### Argument fields

Fields have one unique dot-separated `path`. Paths address object properties;
numeric array indexing, prototype names, and empty path segments are forbidden.
An object field owns its descendant paths. A field is one of `text`, `number`,
`boolean`, `enum`, `object`, `array`, or `structured`.

* `required` means every story and rendered edit must supply the field unless a
  `default` is declared.
* `enum` requires non-empty, unique JSON-scalar `options`.
* `number` may set inclusive `minimum` and `maximum`; `minimum <= maximum`.
* `text` may set `minLength`, `maxLength`, and the supported formats
  `plain-text`, `identifier`, `url`, and `renderable-text`.
* `object` and `array` define child/item schemas. They are never free-form JSON
  by default.
* `structured` is only for the renderer's allowlisted values listed below.

Defaults and story args use the same validation pipeline. Partial form edits
are merged at the edited paths, validated against the complete effective args,
and applied only when valid. The last valid render remains visible on error.
Conditional display uses `visibleWhen: { path, equals }`; it changes form
visibility only, never validation or the public component API.

### Safe values and explicit rejections

Ordinary values are JSON null, booleans, finite numbers, strings, arrays, and
plain objects. Renderable structured values are restricted to `$text`, `$node`,
`$icon`, `$handler`, `$rowKey`, `$columns`, and `$filters`; each has the
existing preview resolver's fixed data shape. Values containing functions,
imports, executable source, prototype keys, cyclic data, `NaN`, `Infinity`, or
an unknown `$` tag are invalid. A diagnostic identifies the source file, JSON
pointer, field path, rule, and human remediation.

## Components, state, and environments

A component story supplies public props only. For an uncontrolled component,
stories select a public default and use an interaction such as `click`, `type`,
`key`, or `focus` to reach real internal state. For a controlled component,
the component exposes its normal `value`/`open` plus change callback contract;
the story wrapper owns the small deterministic controlled-state adapter.

External state is not faked by rewriting imported hooks. Instead, an asset
declares an environment key and fixture ids, and the runtime resolves those ids
from a server-owned registry. Fixtures are data-only and may expose only the
adapter/provider inputs documented by that key. Dynamic imports, arbitrary
providers, production data, storage writes, and network side effects are not
permitted.

```json
{
  "environment": {
    "fixtures": [
      {
        "key": "voiceInput",
        "adapter": "voice-input",
        "options": ["idle", "permission-denied", "recording"]
      }
    ]
  },
  "stories": [{ "id": "denied", "environment": { "voiceInput": "permission-denied" } }]
}
```

## Interactions and expectations

Interactions are ordered, identity-scoped and use only `click`, `type`, `key`,
`focus`, `blur`, `waitFor`, and `settle`. Targets use named selectors or safe
role/name locators. `settle` waits for the declared deterministic idle signal;
timeouts are diagnostics, never implicit success. Expectations use `role`,
`text`, `attribute`, `visible`, or `notVisible`. They cannot execute scripts,
inspect arbitrary browser state, or create network/storage mutations.

## Hook stories

Hook contracts use `kind: "hook"`; their `args` schema describes only the
dedicated fixture harness's declared inputs (for example `active` or `mode`),
not callback, ref, or implementation values. Each story provides valid inputs,
optionally chooses declared adapter fixtures, runs allowlisted hook actions,
settles, and asserts observable DOM output. The hook runner mounts a dedicated
fixture harness; it does not render a fake production component nor mutate hook
internals. `useVoiceInput` is the canonical provider-backed hook: its story
selects a deterministic media/transport fixture and observes state after its
registered start/stop actions.

## Worked stateful example

```json
{
  "schemaVersion": 1,
  "kind": "component",
  "args": { "fields": [{ "path": "open", "kind": "boolean", "default": false }] },
  "environment": { "fixtures": [] },
  "stories": [{
    "id": "open-by-user",
    "name": "Opened from default state",
    "args": { "open": false },
    "interactions": [{ "kind": "click", "target": { "role": "button", "name": "Open" } }, { "kind": "settle" }],
    "expect": [{ "kind": "role", "role": "dialog" }]
  }]
}
```

## Migration invariant

Catalog conformance fails when an eligible version has no valid `story.json`,
has more than one, still contains `examples.json`, declares legacy `controls`
or `setup`, uses an undeclared environment fixture, or declares an invalid
frame. This is a greenfield cutover: there is no compatibility reader.

## Cross-References

- [CODE: api/internal/components/indexer.go]
- [CODE: api/handlers/preview/static.go]
- [DOC: DATA.md]
- [DOC: FLOWS.md]
