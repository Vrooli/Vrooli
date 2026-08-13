# Proto Codegen Pipeline

`path:packages/proto/` is the single source of truth for Vrooli's protobuf schemas and the multi-language code they generate. This page describes how the pipeline is wired and how to make routine changes (add a schema, refresh a vendored module, bump a plugin).

## TL;DR

```bash
cd packages/proto
make generate SCENARIO=<scenario>  # staged, locked, dependency-aware routine path
make lint       # buf lint
make breaking SCENARIO=<scenario>  # proto-health impact check
make verify-committed-gen  # CI-style committed gen/ diff guard
make refresh-vendor  # refresh vendor/ from BSR (rare, requires login)
```

Routine schema edits use a scoped invocation so one change does not rewrite the
fleet. All of `make generate`, `make lint`, `make breaking`, and
`make verify-committed-gen` are **fully offline-capable** — zero outbound BSR
requests. `make refresh-vendor` is the only target that contacts buf.build (see
[Refreshing vendored modules](#refreshing-vendored-modules)). Generation uses
the Go `protogen` command, an advisory cross-process lock, a staging tree beside
`gen/`, and publish-on-change semantics. Scoped runs include reverse dependents
and shared imports.

Use a full-fleet generation only deliberately, for example after a plugin or
vendored-module change:

```bash
cd packages/proto
make generate                 # deliberate full-fleet rebuild
make verify-committed-gen
```

## Pipeline shape

```
schemas/                            ← source .proto files
vendor/googleapis/                  ← vendored BSR module (workspace member)
vendor/protovalidate/               ← vendored BSR module (workspace member)
buf.yaml                            ← lists 3 workspace modules
buf.gen.yaml                        ← invokes 6 plugins on schemas/
gen/{go,typescript,typescript/js,python}/   ← committed output
gen/descriptor/image.binpb                  ← committed descriptor image
gen/manifests/<scenario>.lock.json          ← committed generation manifests
```

`buf.yaml` declares three workspace modules. The vendored BSR modules sit alongside `schemas/` so transitive imports (`buf/validate/validate.proto`, `google/api/annotations.proto`, ...) resolve locally — no BSR fetch.

The `vendor/googleapis/` and `vendor/protovalidate/` trees are required source
inputs and must be committed with the Proto package. They are explicitly
unignored in the repository because a fresh clone and Bridge working-tree
transfer must carry them. Build outputs such as `gen/`, UI `dist/`, and
`node_modules/` remain generated and are intentionally rebuilt by setup.

`buf.gen.yaml` runs six plugin invocations against the `schemas/` input:

| Plugin reference | Output | Plugin binary |
|---|---|---|
| `local: protoc-gen-go` | `path:packages/proto/gen/go/` | `protoc-gen-go` (installed via `go install`, pinned in `internal/tools/protoc-gen-go/tool.json`) |
| `local: protoc-gen-connect-go` | `path:packages/proto/gen/go/` | `protoc-gen-connect-go` (installed via `go install`, pinned in `internal/tools/protoc-gen-connect-go/tool.json`) |
| `local: protoc-gen-es` (×2: ts + js) | `path:packages/proto/gen/typescript/`, `path:packages/proto/gen/typescript/js/` | `@bufbuild/protoc-gen-es` (npm, pinned in `internal/tools/protoc-gen-es/tool.json`). Protobuf-ES v2 emits service descriptors consumed by Connect-ES v2; no separate Connect-ES plugin is used. |
| `protoc_builtin: python` | `gen/python/*.py` | `protoc` built-in (pinned in `internal/tools/protoc/tool.json`) |
| `protoc_builtin: pyi` | `gen/python/*.pyi` | `protoc` built-in |

All six are local. `remote: buf.build/...` is **forbidden** — enforced by `internal/protocodegen/buf_gen_guard_test.go`.

Why local? Two reasons:

1. **Offline / sovereignty.** A laptop on flight Wi-Fi, a CI runner with egress blocked, an agent box with no BSR account — all generate code identically.
2. **Rate-limit immunity.** BSR's anonymous per-IP rate limit was exhausted in a single iteration session when many concurrent agents ran `buf generate`. Local plugins remove that ceiling entirely.

## Adding a new .proto

1. Create the file under `packages/proto/schemas/<scenario>/v1/<dir>/<name>.proto` matching the existing layout.
2. `cd packages/proto && make generate SCENARIO=<scenario>` (or repeat
   `SCENARIO` for several related edits).
3. Commit `schemas/<...>`, the new files under `gen/`, and the refreshed `gen/manifests/<scenario>.lock.json`.
4. `make verify-committed-gen` enforces the committed generated artifacts match.

No buf.gen.yaml change is needed for a new file — the existing 6 plugin invocations cover whatever protos are under `schemas/`.

## Adding a new plugin

1. Pick a binary that:
   - Has a stable distribution channel (`go install`, npm, pypi, github release tarball).
   - Reads `CodeGeneratorRequest` from stdin and writes `CodeGeneratorResponse` to stdout (the protoc plugin contract).
2. Create `internal/tools/<plugin>/tool.json` declaring the binary, version pin, and platform mappings.
3. Create `internal/tools/<plugin>/handler.go` implementing `hostreqkit.Handler`. Mirror an existing handler:
   - Go-installable binaries: see `internal/tools/protoc-gen-go/handler.go`.
   - Npm packages: see `internal/tools/protoc-gen-es/handler.go`.
   - Native release tarballs: see `internal/tools/buf/handler.go`.
   - Native package-manager installs: see `internal/tools/protoc/handler.go`.
4. Register the handler in `internal/runtime/registry.go` `customToolHandlers` and update `internal/runtime/runtime_test.go`'s expected tool list.
5. Add the plugin to `.vrooli/service.json` `hostTools[]` so `vrooli setup` installs it.
6. Reference the plugin in `packages/proto/buf.gen.yaml` (`local:` form).
7. Run the deliberate full-fleet `make generate` and commit the new `gen/`
   outputs.

## Bumping a plugin version

1. Update both the manifest `version` field (`internal/tools/<plugin>/tool.json#/version`) and the handler's `defaultVersion` constant. The handler test asserts they match.
2. `vrooli setup` (or run the install commands manually) to upgrade the local binary.
3. `cd packages/proto && make generate` (a plugin change intentionally
   restamps the full generated fleet).
4. Commit the manifest, handler, and the resulting `gen/` diff. CI's `make verify-committed-gen` enforces the diff is consistent.

## Generation manifests

The full-fleet form, `make generate`, builds every output in a temporary sibling
tree, rebuilds the
descriptor image and lockfiles there, then publishes changed language files by
atomic per-file rename while preserving their existing directory roots; the
descriptor is published as one atomic file rename. Unchanged outputs are
discarded without changing their modification times. Scoped runs publish only
the selected scenarios, their reverse dependents, shared import outputs, and
the global descriptor.

`packages/proto/gen/` is buf-output only. Keep scenario helpers, compatibility
shims, hand-authored enum maps, debug renderings, and copied vendor files out
of this tree; place those artifacts in the owning scenario or in source
directories that are not committed as generated output. The only non-language
artifacts allowed under `gen/` are `typescript/package.json`,
`descriptor/image.binpb`, and `manifests/*.lock.json`. The JSON descriptor
rendering is intentionally not produced or committed.

Each lockfile is owned by `packages/proto` and records:

- the scenario slug,
- the digest of that scenario's `.proto` source closure, including transitive
  imports resolved from `schemas/` and vendored modules,
- the committed codegen toolchain pins and `buf.gen.yaml` digest,
- every generated output path for that scenario with a SHA-256 digest.

The lockfiles let `proto-health validate scenario <name>` verify generated
artifact sync with file reads and hashes only. proto-health does not run `buf`
or mirror codegen post-steps on its hot path; it trusts the committed manifest
and reports stale inputs, edited outputs, or orphan generated files as
`proto.gen_out_of_sync`. A missing lockfile is
`proto.gen_manifest_missing`, and changed toolchain pins are advisory
`proto.gen_toolchain_drift` until `make generate` refreshes the manifests.

`make verify-committed-gen` is the authoritative boundary gate. It runs the
same staged generation path under the same lock, compares every committed file
under `gen/` (including descriptors and manifests), and leaves the worktree
unchanged. Failed verification retains its staging path in the log for
diagnosis; successful verification removes it.

## Descriptor snapshot contract

Runtime consumers use `packages/proto/descriptorimage.Source`, which watches
the global descriptor and CLI manifests, caches an immutable parsed snapshot,
and revalidates stamps on each request. A successful publication increments the
generation and exposes its digest and load time. A malformed reload keeps the
last known-good snapshot and reports the error through the consumer health
surface; only an initial load failure prevents startup after bounded retries.
Handlers pin one snapshot for an operation, so a request never mixes contract
generations. Program-runtime gives new kernels the current binding projection
while an in-flight kernel retains the generation it started with.

## Refreshing vendored modules

`path:packages/proto/vendor/googleapis/` and `path:packages/proto/vendor/protovalidate/` are exported snapshots of the corresponding BSR modules. Refresh is a deliberate, occasional operation:

```bash
cd packages/proto
buf export buf.build/googleapis/googleapis -o vendor/googleapis
buf export buf.build/bufbuild/protovalidate -o vendor/protovalidate
make generate
make verify-committed-gen
```

The `buf export` calls require BSR access. Anonymous works until the rate limit kicks in; an authenticated session (see [`docs/configuration/integrations/buf-bsr.md`](../configuration/integrations/buf-bsr.md)) raises the ceiling. Login is the **only** time the BSR token matters — codegen itself is unconditional after the refresh.

A `make refresh-vendor` target wraps the above so contributors don't have to memorize the URLs. If the resulting `gen/` diff is unexpected, roll back the vendor change rather than committing drift; `make verify-committed-gen` is the CI-style committed-artifact gate.

Working-tree onboarding validates that these vendor trees are present in the
enumerated source closure before shipping files to a node. If that check fails,
restore the committed snapshots or deliberately run `make refresh-vendor` and
regenerate; do not fix the problem by transferring `dist/` or `node_modules/`.

## Troubleshooting

### "Cannot read properties of undefined" from protoc-gen-es

The protoc-gen-es v2 series shipped a parsing regression in early 2.x patches (e.g. 2.5.x). Ensure the installed version matches the manifest pin (currently `2.12.0`). `protoc-gen-es --version` should print the pinned value; `npm install --prefix ~/.cache/vrooli/protoc-plugins/node @bufbuild/protoc-gen-es@<pinned>` re-installs.

### `make verify-committed-gen` shows comment-only diffs after a plugin upgrade

Any plugin version bump (`protoc-gen-go`, `protoc-gen-connect-go`, `protoc-gen-es`, `protoc`) re-stamps the generator-version comment at the top of every generated file. Commit the diff alongside the manifest pin bump in the same PR. There is no special-casing of comment-only diffs — the version is part of the plugin's output contract.

### "machine buf.build" is missing from ~/.netrc

`buf registry login` writes the entry. If the CLI is signed in but the line is missing, the user-name on the `machine` line is wrong (multi-account systems sometimes layer entries). Run `buf registry logout && buf registry login` to reset.

### Plugin not on PATH after install

The plugin handlers symlink the installed binary into `~/.local/bin/`. Vrooli's setup ensures `~/.local/bin/` is on PATH. If a plugin is installed but `which protoc-gen-X` fails, manually source the shell rc that exports `~/.local/bin` (or restart the terminal).

### Codegen produces a diff under `gen/` after a no-op edit

Re-run `make generate`. The committed `gen/` tree plus
`gen/manifests/*.lock.json` is the canonical baseline; any divergence is either
(a) a plugin version drift, (b) a vendored-module drift, (c) an in-flight schema
edit, or (d) an orphan generated file pruned by the clean generation step.
`git status` against `gen/` distinguishes the cases.

## See also

- [`../configuration/host/tools.md`](../configuration/host/tools.md) — host-tool model that backs the plugins
- [`../configuration/integrations/buf-bsr.md`](../configuration/integrations/buf-bsr.md) — optional BSR sign-in (vendor refresh only)
- [`../../packages/proto/buf.yaml`](../../packages/proto/buf.yaml), [`../../packages/proto/buf.gen.yaml`](../../packages/proto/buf.gen.yaml) — pipeline source-of-truth
