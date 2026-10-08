# Test isolation contract

Storage-manager has independent persistence seams:

- SQLite is opened through `database.Open`, initialized with
  `database.EnsureSchemas`, marked by `apihttp.TestModeMiddleware`, and
  exposed to test-genie through `devrouting.RegisterWithFileRoots`.
- File state is rooted by `filerouting.New(primaryPaths)`. File stores select
  their class root with `RoutedRoots.Pick(ctx, class)` for every request and
  record successful writes with `RoutedRoots.RecordWrite(ctx)`.

During a test-genie lease, the request context selects disposable test roots
and the routed database selects the disposable test pool. Production roots
remain the fallback for ordinary requests and for startup reads. A file
store must not cache a path selected from a test request or write directly to
an ambient home, repository, or temporary path.

The `FILE_ROUTED_SEAMS_UNWIRED` and `ROUTED_SEAMS_UNWIRED` analyzers are the
static gate for this contract. A scenario with either finding cannot run
mutating E2E playbooks safely.
