# Proto Codegen Pipeline

`path:packages/proto/` is the single source of truth for Vrooli's protobuf schemas and the multi-language code they generate. This page describes how the pipeline is wired and how to make routine changes (add a schema, refresh a vendored module, bump a plugin).

## TL;DR

```bash
cd packages/proto
make generate   # regenerate gen/ (Go, TS, JS, Python, pyi)
make lint       # buf lint
make breaking   # buf breaking-change check vs base branch
make check      # generate + diff guard (used in CI)
make refresh-vendor  # refresh vendor/ from BSR (rare, requires login)
```

All of `make generate`, `make lint`, `make breaking`, `make check` are **fully offline-capable** — zero outbound BSR requests. `make refresh-vendor` is the only target that contacts buf.build (see [Refreshing vendored modules](#refreshing-vendored-modules)).

## Pipeline shape

```
schemas/                            ← source .proto files
vendor/googleapis/                  ← vendored BSR module (workspace member)
vendor/protovalidate/               ← vendored BSR module (workspace member)
buf.yaml                            ← lists 3 workspace modules
buf.gen.yaml                        ← invokes 6 plugins on schemas/
gen/{go,typescript,typescript/js,python}/   ← committed output
```

`buf.yaml` declares three workspace modules. The vendored BSR modules sit alongside `schemas/` so transitive imports (`buf/validate/validate.proto`, `google/api/annotations.proto`, ...) resolve locally — no BSR fetch.

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
2. `cd packages/proto && make generate`.
3. Commit `schemas/<...>` and the new files under `gen/`.
4. `make check` enforces the diff matches.

No buf.gen.yaml change is needed for a new file — the existing 5 plugin invocations cover whatever protos are under `schemas/`.

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
7. Run `make generate` and commit the new `gen/` outputs.

## Bumping a plugin version

1. Update both the manifest `version` field (`internal/tools/<plugin>/tool.json#/version`) and the handler's `defaultVersion` constant. The handler test asserts they match.
2. `vrooli setup` (or run the install commands manually) to upgrade the local binary.
3. `cd packages/proto && make generate`.
4. Commit the manifest, handler, and the resulting `gen/` diff. CI's `make check` enforces the diff is consistent.

## Refreshing vendored modules

`path:packages/proto/vendor/googleapis/` and `path:packages/proto/vendor/protovalidate/` are exported snapshots of the corresponding BSR modules. Refresh is a deliberate, occasional operation:

```bash
cd packages/proto
buf export buf.build/googleapis/googleapis -o vendor/googleapis
buf export buf.build/bufbuild/protovalidate -o vendor/protovalidate
make generate
make check
```

The `buf export` calls require BSR access. Anonymous works until the rate limit kicks in; an authenticated session (see [`docs/configuration/integrations/buf-bsr.md`](../configuration/integrations/buf-bsr.md)) raises the ceiling. Login is the **only** time the BSR token matters — codegen itself is unconditional after the refresh.

A `make refresh-vendor` target wraps the above so contributors don't have to memorize the URLs. If the resulting `gen/` diff is unexpected, roll back the vendor change rather than committing drift; `make check` is the gate.

## Troubleshooting

### "Cannot read properties of undefined" from protoc-gen-es

The protoc-gen-es v2 series shipped a parsing regression in early 2.x patches (e.g. 2.5.x). Ensure the installed version matches the manifest pin (currently `2.12.0`). `protoc-gen-es --version` should print the pinned value; `npm install --prefix ~/.cache/vrooli/protoc-plugins/node @bufbuild/protoc-gen-es@<pinned>` re-installs.

### `make check` shows comment-only diffs after a plugin upgrade

Any plugin version bump (`protoc-gen-go`, `protoc-gen-connect-go`, `protoc-gen-es`, `protoc`) re-stamps the generator-version comment at the top of every generated file. Commit the diff alongside the manifest pin bump in the same PR. There is no special-casing of comment-only diffs — the version is part of the plugin's output contract.

### "machine buf.build" is missing from ~/.netrc

`buf registry login` writes the entry. If the CLI is signed in but the line is missing, the user-name on the `machine` line is wrong (multi-account systems sometimes layer entries). Run `buf registry logout && buf registry login` to reset.

### Plugin not on PATH after install

The plugin handlers symlink the installed binary into `~/.local/bin/`. Vrooli's setup ensures `~/.local/bin/` is on PATH. If a plugin is installed but `which protoc-gen-X` fails, manually source the shell rc that exports `~/.local/bin` (or restart the terminal).

### Codegen produces a diff under `gen/` after a no-op edit

Re-run `make generate`. The committed `gen/` tree is the canonical baseline; any divergence is either (a) a plugin version drift, (b) a vendored-module drift, or (c) an in-flight schema edit. `git status` against `gen/` distinguishes the cases.

## See also

- [`../configuration/host/tools.md`](../configuration/host/tools.md) — host-tool model that backs the plugins
- [`../configuration/integrations/buf-bsr.md`](../configuration/integrations/buf-bsr.md) — optional BSR sign-in (vendor refresh only)- [`../../packages/proto/buf.yaml`](../../packages/proto/buf.yaml), [`../../packages/proto/buf.gen.yaml`](../../packages/proto/buf.gen.yaml) — pipeline source-of-truth
