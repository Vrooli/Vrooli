# Phase 10 surviving regular expressions

The structural facts index now owns JSX element, attribute, export, import,
call, and inline-style observations. The expressions below remain because
they validate text contracts rather than syntax trees.

| Location | Expression family | Reason it remains textual |
|---|---|---|
| `api/internal/gates/overlay-surface-composition.go` | CSS policy markers and provenance stamps | These rules inspect authored CSS and documentation metadata, not AST nodes. |
| `api/internal/gates/api.go` | User-facing copy and legacy translation markers | The contract is the presence of literal text and semantic string keys. |
| `api/internal/gates/specifier-shape.go` | Published package specifier grammar | This is the shared string grammar, already covered by the specifier fixture table. |
| `api/internal/gates/version-liveness.go` | Imported package specifiers | The rule compares import strings to released version identities. |
| `api/internal/gates/fallback-parity.go` | CSS variable declarations | The fallback contract is declared in stylesheet text. |
| `api/internal/gates/conformance.go` | Utility and token vocabulary | These expressions recognize forbidden textual utility spellings. |
| `api/internal/gates/composition-contract.go` | Elevation token references | The rule compares CSS token text against the composition contract. |
| `api/internal/gates/field_ownership.go` | JSDoc ownership tags | The rule is specifically about authored documentation text. |
| `api/internal/gates/documentation.go` | Exported symbol documentation | The rule checks a source file's documentation boundary. |
| `api/internal/gates/tokencensus.go` | Token declaration syntax | Token census reads CSS declaration text. |

No expression in this record is used as a substitute for JSX traversal. New
structural rules must extend `resolve-imports.mjs` facts and its fixtures.

The `ResponsiveDialog/1.2.9` stylesheet migration used the generator's
explicit `RCL_ACCEPT_RELEASE_MIGRATION=1` mode to refresh its historical hash
after changing the release shape. Ordinary generator runs continue to treat
those bytes as immutable.
