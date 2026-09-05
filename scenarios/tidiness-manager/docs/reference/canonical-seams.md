# Canonical Seam Rules

Use `.vrooli/canonical-seams.json` to declare one reviewed implementation for a job and the Go
syntax that bypasses it. Tidiness Manager parses Go with `go/ast`; comments and documentation
examples do not produce findings.

Each seam requires `id`, `canonical`, `why`, `remediation`, `bypass`, `scope`, `severity`, and
`budget`. Scope globs are relative to the scanned target. A positive `budget` or `reserve`
requires `reserve_reason`. The scanner emits `BYPASSED_SEAM` only for observations above the
allowance.

Accepted bypass kinds: `call`, `literal`, `declaration`, `semantic-naming`, `replaced-call`, `shape`, `directive`, `suppression-breadth`, `absence`.

## Bypass kinds

### `call`

Matches the resolved name of an `ast.CallExpr`, including import aliases. Required field:
`pattern`. Example: `{"kind":"call","pattern":"^os\\.Rename$"}`.

### `literal`

Matches an unquoted string or numeric literal. A direct argument can be selected with
`<resolved-call-name>:<value>` and a duration multiplication with
`<resolved-selector-name>:<numeric-value>`. Required field: `pattern`. Example:
`{"kind":"literal","pattern":"^time\\.Second:[0-9]+$"}`.

### `declaration`

Matches declared symbols. Required fields: `pattern`, `declKind`; supported declaration kinds
are `func`, `method`, `type`, `interface`, `const`, and `var`. Optional
`repeatedAcrossPackages` sets a package-count threshold, `excludeSymbols` removes exact
symbol names, and `excludeAliases` omits `type X = Y`. `pairedPathPatterns` accepts exactly two
regular expressions with one capture group each; a declaration matches only when the same
captured key contains the symbol on both sides. Example:
`{"kind":"declaration","pattern":"^Test[A-Za-z]+$","declKind":"func","repeatedAcrossPackages":3,"excludeSymbols":["TestConformance"]}`.

### `semantic-naming`

Evaluates declarations selected by `declKind`. It splits identifiers at underscores, digits,
camel-case, and acronym boundaries. Single letters, digit-only words, and words in
`genericWords` do not carry domain meaning. A declaration matches when its name contains fewer
than `minDomainWords` other words. The blank identifier is ignored. Required fields: `pattern`,
`declKind`, `minDomainWords`; optional field: `genericWords`. Example:
`{"kind":"semantic-naming","pattern":".*","declKind":"const","genericWords":["value"],"minDomainWords":1}`.

### `replaced-call`

Detects an unexported helper that reimplements the operation named by `canonicalCall`.
Required fields: `pattern`, `canonicalCall`, `shapeKind`. Example:
`{"kind":"replaced-call","pattern":".*","canonicalCall":"slices.Sort","shapeKind":"nested_swap_loop"}`.
For `nested_swap_loop`, calls other than structural built-ins `len` and `cap` exclude a helper;
exported declarations are also excluded.

### `shape`

Matches one of the structural predicates below. Required fields: `pattern`, `shapeKind`, plus
the fields required by that shape. Example:
`{"kind":"shape","pattern":".*","shapeKind":"json_nesting","outerKey":"dependencies","innerKey":"resources"}`.

### `directive`

Matches the trimmed text of a Go comment directive. Required field: `pattern`. Example:
`{"kind":"directive","pattern":"^//nolint:gocyclo$"}`.

### `suppression-breadth`

Matches linter directives and measures their coverage. Required field: `pattern`; optional
`unit` is `declarations` (default) or `directives`. A file-level directive costs the number of
non-import top-level declarations, while a declaration- or statement-level directive costs
one. Example:
`{"kind":"suppression-breadth","pattern":"^//nolint:goconst","unit":"declarations"}`.

### `absence`

Declares a target that must be present, or an artifact kind that must be absent. Required
fields: `pattern`, `requireFor`, and either `requirePresent` or `forbidKind`. The supported
`forbidKind` is `executable`. Example:
`{"kind":"absence","pattern":"^$","requireFor":"**/service.json","requirePresent":"schema.json"}`.

## Shape kinds

Accepted shape kinds: `switch_on_argv`, `interface_method_set`, `struct_field_set`, `error_boundary`, `context_duration_literal`, `json_nesting`, `constructs_type`, `nested_swap_loop`.

- `switch_on_argv` matches a switch on argument index zero. It requires no extra fields.
- `interface_method_set` groups equivalent interfaces; optional `minMembers` defaults to two.
- `struct_field_set` groups equivalent structs; optional `minMembers` defaults to two.
- `error_boundary` matches a named function that returns `error` and calls an `.Error` method.
- `context_duration_literal` matches a configured context call whose duration contains a
  numeric literal.
- `json_nesting` matches an anonymous struct with JSON fields nested from required `outerKey`
  to required `innerKey`.
- `constructs_type` matches a function that constructs required `constructedType`, expressed as
  a package-qualified Go type such as `structpb.Value`.
- `nested_swap_loop` matches nested `for` loops whose inner loop swaps two elements of the same
  slice. It is used by `replaced-call`.

## Resolution and overlays

The resolver merges files in this order:

1. the repository baseline at `<repo-root>/.vrooli/canonical-seams.json`;
2. the optional target overlay at `<target-root>/.vrooli/seams.json`.

Later entries replace earlier entries by `id` as complete seam declarations. Fields are never
merged individually. An overlay can add a new id or replace a baseline id. It can disable a
baseline seam with `{"id":"baseline-id","disabled":true,"reserve_reason":"why this target is not applicable"}`.
A disable without `reserve_reason`, or a disable naming no baseline seam, is rejected. Missing
overlay files are normal. Every scan returns `seam_files` in merge order, including only files
that were present and loaded.

## Validation

Run from `scenarios/tidiness-manager/api`:

```bash
go test ./... -run 'Test(ScanSeams|LoadSeams|RepositoryCanonical|Seam|Doc)' -count=1
```
