# Scenario freshness — keyed build inputs

The scenario freshness engine (`internal/lifecycle/freshness.go`,
`packages/cli-core/cliutil/manifest.go`) decides whether a scenario's compiled
artifact must be rebuilt. It hashes the exact set of source files a binary
imports (the precise `go list -deps` closure) — so an edit to an *unrelated*
scenario never marks this one stale.

The manifest engine is the only freshness verdict authority. It enumerates
inputs with `git ls-files --cached --others --exclude-standard` inside a work
tree and uses a conservative `WalkDir` fallback outside one. Git only selects
candidate files; it never decides whether an artifact is stale. A missing or
invalid manifest is stale once and is stamped after the next successful build.
There is no mtime-only fallback.

For Go components, deriving the precise import closure is cached beside the
component at `.vrooli-closure-go_module.json`. The cache key includes the
component's `go.mod` and `go.sum`, every local replacement module's module
files, and the resolved Go toolchain version. A warm freshness check therefore
does not rerun `go list`; changing any of those inputs invalidates the sidecar
and refreshes the closure. The sidecar is a build output and is excluded from
all freshness manifests.

A pure content hash has one blind spot: when a **non-file build input** changes
(toolchain, target OS/arch, CGO, build tags) the source bytes are identical, so a
content hash reports *fresh* even though a rebuild would emit a different binary.
The engine closes that false negative by folding a curated set of
output-determining scalars into the manifest digest as `key_inputs`, alongside
the file fingerprints.

## The contract

`key_inputs` is a `map[string]string`. It **only ever grows** — the file input
set is never touched by this mechanism, which is the mechanical guarantee that
keying build inputs cannot reintroduce the false-positive restart cascade the
engine was rewritten to eliminate.

### Binaries (`binaries` check)

Resolved by `goBuildKeyInputs` from the **resolved build environment** (`go env`,
which reflects toolchain defaults + overrides actually in effect — not raw
`os.Getenv`, which is blank when a default would still apply) plus the scenario's
declared `go build` step:

| Key | Source | Why it determines output |
|---|---|---|
| `toolchain` | `go version` | codegen differs across Go releases |
| `goos`, `goarch` | `go env` | selects the target platform |
| `cgo_enabled` | `go env` | changes linkage for cgo-conditional packages |
| `goamd64`, `goarm` | `go env` | sub-arch ISA level |
| `goflags`, `goexperiment` | `go env` | toolchain-wide codegen flags |
| `build_tags` | `-tags` on the setup `go build` step | selects build-constrained files |
| `ldflags` | `-ldflags` on the setup step | injects symbols / strip flags |
| `trimpath` | `-trimpath` on the setup step | rewrites embedded paths |

### UI (`ui-bundle` check)

Resolved by `uiBuildKeyInputs`:

| Key | Source | Why |
|---|---|---|
| `node_env` | `NODE_ENV` | Vite emits different output for dev vs prod |
| `node_major` | `node --version` major | esbuild/Vite output can differ across Node majors |

### Other registered builders

Builder knowledge lives in the lifecycle registry. The `python_uv` row keys
`pyproject.toml`, `uv.lock`, Python, and uv versions, and stamps
`api/.venv/pyvenv.cfg`. Component verdicts are evaluated independently, so a
fresh UI does not rebuild because an API is stale and vice versa.

## Rules

- **Curated allowlist only.** A var is keyed *only* if a change in it changes the
  compiled bytes. No `os.Environ()` sweep; volatile vars (`$TERM`, `$PWD`, …) are
  never keyed — keying them would manufacture false positives.
- **Resolved, not ambient.** Values come through the `hostProbeDeps.goEnv` /
  `nodeVersion` seams (injectable in tests), never raw `os.Getenv` in decision
  logic, and never a `runtime.GOOS` branch.
- **Omit-on-unknown.** An unresolvable determinant is dropped, never defaulted.
  Bias: accept a rare exotic false negative over any new false positive.
- **Normalized.** Tag lists are split/sorted/rejoined so ordering noise
  (`-tags b,a` vs `-tags "a b"`) never flaps the digest; empty values are omitted
  so present-empty and absent are identical.

## What you see

`vrooli scenario freshness <name> --explain` names a changed build input as
precisely as a changed file:

```
api/foo-api stale (build input changed: cgo_enabled): cgo_enabled
```

Adding a key is backward-compatible: a manifest stamped before the key existed
reads stale **once** on the next check (a present key the recorded set lacks),
then re-stamps on rebuild. No manifest version bump is required.

## Tests

- `internal/lifecycle/freshness_buildinput_test.go` — file-set invariance (the
  guardrail), per-key false-negative closure with `--explain` naming,
  omit-on-unknown, tag/empty normalization, build-command parsing and binding.
- `internal/lifecycle/freshness_integration_test.go` — unrelated-edit-stays-fresh,
  imported-edit-rebuilds, toolchain/determinant change ⇒ stale (real `go env`).
