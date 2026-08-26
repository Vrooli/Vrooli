# Story Contract v5

`story.json` is the single declarative preview and test contract for one
catalog asset version. Version 5 is the authored contract: it carries public
argument definitions, deterministic fixtures, named composition roles, story
states, interactions, expectations, and evidence intent. Executable preview
recipes live only in the version-local `story.tsx`.

Each story names the question it answers: `anatomy` is the default rendered
shape, `axis` is a matrix for one declared enum prop, and `boundary` demonstrates
a meaningful edge condition. Axis stories declare their prop and the rendered
option union in `covers`; boundary stories may name states such as `empty`,
`truncate`, `overflow`, `loading`, `error`, `disabled`, `rtl`, or `forced-colors`.

## Required shape

Every contract has `schemaVersion: 5`, `kind` (`component` or `hook`), `args`,
`environment.fixtures`, and at least one named story. Unknown fields are errors.
The published machine-readable contract is
`.vrooli/schemas/story-contract.schema.json`.

```json
{
  "$schema": "../../../../.vrooli/schemas/story-contract.schema.json",
  "schemaVersion": 5,
  "kind": "component",
  "args": { "fields": [{ "path": "tone", "kind": "enum", "options": ["success", "warning"] }] },
  "environment": { "fixtures": [] },
  "composition": {
    "specimen": { "module": "./story.tsx", "export": "MetricCard" },
    "fixture": { "asset": "fixtures.resource-collection", "version": "1.0.0", "state": "ready" },
    "frame": { "asset": "navigation.page", "version": "1.0.0", "region": "content", "fixture": "fixtures.user-directory" }
  },
  "stories": [{
    "id": "success",
    "name": "Success",
    "role": "axis",
    "axis": "tone",
    "covers": { "tone": ["success", "warning"] },
    "args": { "tone": "success" },
    "interactions": [{ "kind": "settle" }],
    "expect": [{ "kind": "role", "role": "status" }],
    "evidence": { "reviewSet": "core", "states": ["ready"] }
  }]
}
```

`composition.specimen` is a named export from `./story.tsx`. `composition.harness`
is a versioned catalog harness with `asset`, `version`, `export`, and optional
JSON `config`. `composition.fixture` and `composition.frame` are immutable
catalog references. A story-level composition overrides only the roles it
declares; there is no second story-level `frame` field.

## Arguments and safe values

Each argument field has one unique dot-separated path and one kind: `text`,
`number`, `boolean`, `enum`, `object`, `array`, or `structured`. Values are
JSON scalars, arrays, and plain objects. Structured values use only the
allowlisted renderer tags (`$text`). Component-specific composition belongs in
the version-local `story.tsx`, referenced by `composition.specimen`. Contracts
must not use raw `className` values; style choices belong to the component
token contract. `raw_class_name` is a non-blocking diagnostic with a pointer
and replacement guidance for authored fixtures that still contain that shape.
The removed structured tags `$node`, `$icon`, `$handler`, `$rowKey`,
`$columns`, and `$filters` are blocking parser diagnostics; move their
behavior into the named `story.tsx` specimen instead.

The component API owns naming and accessibility. A bare `aria-*` argument is a
warning that the component should expose the appropriate naming prop instead.

## Story taxonomy and coverage

Every contract has exactly one `role: anatomy` story. Each enum field in
`args.fields` has an `role: axis` story naming that field; its `covers` values
discharge release coverage because they describe what the specimen renders,
not merely the input passed to it. Boundary stories describe conditions that
cannot be reduced to an enum, such as empty content or truncation. Every story
has at least one expectation, and sibling frames must answer distinct rendered
questions. The catalog gates `story-grammar` and `story-distinctness` enforce
these rules.

## Derived requirements

Selector identity and i18n keys are derived data, not authored story or
manifest declarations. The selector id is the stable catalog asset id and is
emitted by the selector-coverage derivation alongside its source pointer. The
i18n gate writes the derived key set to the generated catalog coverage record,
including the source hash used to detect staleness. A changed source hash
invalidates the record and requires derivation before adoption.

Token requirements follow the same rule: the indexer derives external CSS
custom properties from version source. `requiredTokens` is not a manifest
field.

## Deterministic behavior

Stories provide public props only. Interactions are limited to `click`, `type`,
`key`, `focus`, `blur`, `waitFor`, and `settle`; targets are selectors or safe
role/name locators. Fixtures are data-only references resolved by the
server-owned registry. Expectations use the allowlisted role, text, attribute,
visibility, layout, and count vocabularies. Contracts cannot execute scripts,
import arbitrary providers, write storage, or use production network data.

## Diagnostics and authoring

The parser returns errors and warnings together. Errors block indexing and
execution. Warnings never block and identify authoring debt with a stable rule,
JSON pointer, and one-line remediation. Current normative warning rules are:

- `legacy_raw_node` → named `composition.specimen` in the local `story.tsx`.
- `raw_class_name` → component token contract.
- `aria_prop_workaround` → a real component naming prop/API.

Use `react-component-library components index` or
`react-component-library components stories` to see the warning count and
remediation list. The parser accepts only schemaVersion 5; invalid or obsolete
shapes are corrected in their source contract. The repository contains no
compatibility parser or post-release story rewrite command.

## Component manifest

`component.json` owns catalog identity, release pointers, dependencies, slot,
tags, and design-style affinities. Its published schema is
`.vrooli/schemas/component-manifest.schema.json`. Selector ids, i18n keys, and
token requirements are generated from source and are never duplicated as
hand-authored manifest fields.

## Cross-references

- [CODE: api/internal/components/story_contract.go]
- [CODE: api/internal/components/indexer.go]
- [DOC: guides/asset-preview-composition.md]
