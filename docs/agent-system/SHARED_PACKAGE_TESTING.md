# Shared Package Testing

Shared Go packages own the canonical test assets for their own public
surfaces. A package `<pkg>` may ship a top-level sibling package
`<pkg>test` with fakes, harnesses, and assertion helpers for code that
consumes `<pkg>`.

This mirrors Go's standard-library pattern: `httptest`, `iotest`, and
`fstest` are sibling packages, not nested `test` directories or broad
utility bags.

## Naming

- Use a top-level sibling package named `<pkg>test`.
- Do not create `<pkg>/test/` subpackages.
- Do not create umbrella packages such as `api-core/testing` or
  `cli-core/testkit`.
- Do not use `*kit`, `*util`, or broad helper suffixes for canonical test
  companions.

Canonical import paths look like:

```go
import databasetest "github.com/vrooli/api-core/databasetest"
import connectxtest "github.com/vrooli/api-core/connectxtest"
import cliapptest "github.com/vrooli/cli-core/cliapptest"
```

Aliases are optional, but using the package name as the alias keeps test code
easy to scan.

## Content Rules

Put exported, consumer-facing test helpers in `<pkg>test`:

- Recording fakes for public interfaces, such as
  `databasetest.FakeExecer`.
- Server harnesses for package-owned registration surfaces, such as
  `connectxtest.StartTestServer`.
- Test-only constructors, such as `connectxtest.NewLogger`.
- Assertion helpers that encode package-specific behavior.

Keep production-quality alternate implementations in the production package
itself. If an implementation is a valid runtime choice, it is not test-only.
For example, `blobstore.Memory` stays in `blobstore` because in-memory storage
can be useful outside unit tests.

## Two-Layer Rule

Use the owning boundary to decide where test utilities live:

- Helpers exported for consumers of a shared package live in `<pkg>test`.
- Helpers used only by the shared package's own tests stay close to those
  tests: unexported helpers in `_test.go`, fixtures in `testdata/`, or a
  module-local `internal/testutil` only when repeated across several internal
  packages.

Do not mirror scenario-local UI-style `test-utils` folders in Go shared
packages. The conceptual rule is the same, but the Go import-path shape is
per-package sibling ownership.

## TypeScript Equivalent

When TypeScript shared packages adopt this convention, use subpath exports:
`@vrooli/<pkg>/testing`. The mechanism is package-manager-specific, but the
shape is the same: a package-owned test companion import path.

No TypeScript shared-package test companion is landed by this pilot.

## Per-Package Status

| Module | Package | Status | Reason |
| --- | --- | --- | --- |
| `api-core` | `database` | `databasetest` | Canonical fake for `SchemaExecer`. |
| `api-core` | `connectx` | `connectxtest` | Canonical Connect service server and logger harnesses. |
| `api-core` | `blobstore` | Inline alternate implementations | `Memory` and `Filesystem` are runtime implementations, not test-only fakes. |
| `api-core` | `discovery`, `health`, `pathfilter`, `preflight`, `retry`, `scenario`, `scenariocli`, `secrets`, `server`, `staleness`, `storage` | No companion yet | No exported consumer fake or harness has crossed the extraction threshold. Add `<pkg>test` when a concrete repeated consumer need appears. |
| `cli-core` | `cliapp` | `cliapptest` | Convention-compliant home for test `RunContext` constructors. Existing `cliapp` exports remain. |
| `cli-core` | `cliutil` | No companion yet | Current seams are concrete helpers with direct injection points, especially `HTTPClientOptions.Client`; no exported interface currently justifies a canonical fake. |
| `cli-core` | `buildinfo`, `cmd`, `sandbox-resolve` | No companion yet | No repeated consumer-facing test helper has been identified. |

Update this table whenever a shared package adds a companion.

## Anti-Patterns

Avoid these patterns in shared packages, templates, and scenarios:

- Hand-rolling a `Fake<X>` for an `api-core` or `cli-core` interface when a
  `<pkg>test` fake exists.
- Putting recording fakes in the production package when they are not valid
  runtime implementations.
- Creating umbrella `testing`, `testkit`, or `testutil` packages for unrelated
  helper surfaces.
- Adding compatibility shims in templates instead of consuming the canonical
  companion package directly.

## Examples

Before, a scenario that tests schema application might define a local fake:

```go
type fakeExecer struct {
    queries []string
}

func (f *fakeExecer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
    f.queries = append(f.queries, query)
    return nil, nil
}
```

After, the scenario imports the package-owned fake:

```go
execer := &databasetest.FakeExecer{}
err := database.EnsureSchemas(ctx, execer, providerA, providerB)
queries := execer.SnapshotQueries()
```

Before, a Connect handler test might mount a router and server in every domain
test file. After, the domain test keeps only domain-specific wiring:

```go
path, handler := notesconnect.NewNotesServiceHandler(notes.NewConnectHandler(deps))
server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
client := notesconnect.NewNotesServiceClient(server.Client(), server.URL)
```

