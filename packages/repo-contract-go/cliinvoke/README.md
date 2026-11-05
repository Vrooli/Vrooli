# cliinvoke

`github.com/vrooli/repo-contract-go/cliinvoke` is the only place that resolves, runs, and
classifies a `vrooli` CLI subprocess.

- `Resolve(ResolveOptions)` finds the binary in one fixed order: explicit path,
  `VROOLI_BIN`, the runtime home's `bin` entry from the repo contract, `PATH`.
  A miss returns `*BinaryMissingError` naming every candidate tried.
- `Run(ctx, Invocation)` executes with a deadline and `WaitDelay`, the
  discipline the 2026-08-01 inherited-pipe outage taught. It invokes the binary
  directly on every platform; there is no shell wrapper.
- `Result.Class` is one of `ok`, `usage`, `binary-missing`, `timeout`,
  `refusal`, `lifecycle`. Supervisors retry `timeout` and `lifecycle` only.

The package imports only the standard library and the rest of `repo-contract-go`. Keep it
that way: the autoheal loop refuses the proto and api-core dependency graph so
a drift there cannot take the recovery path down with it.

The conformance test in `conformance_test.go` fails when any Go file under
`internal/`, `cmd/`, `packages/`, or `scenarios/vrooli-autoheal/` spawns
`"vrooli"` through `os/exec` directly.
