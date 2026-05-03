# Proto Codegen Local Plugins + BSR Login Onboarding — Implementation Plan

## 1. Purpose

Make Vrooli's Protobuf codegen **fully offline-capable** and **rate-limit-immune** by switching `buf.gen.yaml` from BSR remote plugins to local plugins managed by Vrooli's host-tool registry, and **wire BSR login awareness into the documented onboarding contract** so that operators on a fresh clone are guided to the (now optional) login step through the same surfaces that already handle Claude Code, Codex, and Cloudflared sign-ins.

Two outcomes:

1. `buf generate` runs without contacting buf.build for codegen plugins — works on a plane, on a GPU box behind a firewall, or with 50 concurrent agents.
2. The `buf registry login` step is declared in the configuration substrate (`docs/configuration/integrations/`) so that when the `vrooli-onboarding` v2 rework picks up integrations, BSR login is surfaced exactly the same way as other `external_sign_in_command` integrations — no special-case code, no bespoke wizard step.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring
```

Source-of-truth context for execution:

```bash
# Configuration substrate the onboarding wizard implements
sed -n '1,200p' docs/configuration/host/tools.md
sed -n '1,200p' docs/configuration/integrations/README.md
sed -n '1,130p' docs/configuration/integrations/external-auth.md
sed -n '1,100p' docs/configuration/integrations/connectors.md

# Onboarding scenario contract (V2 rework picks this up)
sed -n '1,80p' scenarios/vrooli-onboarding/PRD.md
sed -n '60,200p' scenarios/vrooli-onboarding/docs/WIZARD_FLOW.md

# Existing host-tool handler patterns to mirror
sed -n '1,220p' internal/tools/cloudflared/handler.go
sed -n '1,80p' internal/runtime/registry.go
sed -n '1,80p' internal/hostreqkit/manifest.go

# Current proto pipeline
cat packages/proto/buf.gen.yaml
cat packages/proto/buf.yaml
cat packages/proto/Makefile
cat internal/tools/buf/tool.json
sed -n '780,930p' internal/cli/scenariohandlers/template_runtime.go
```

## 3. Hard Rule: Greenfield, No Compatibility Shims

This is a **greenfield switch**, not a migration. Per project guidance:

- **Delete** `scripts/migrate_candidates/tools/buf.sh` once the Go handler lands. Do not preserve it as a fallback or leave a "use this if Go handler fails" path.
- **Replace** every `remote: buf.build/...` plugin entry in `buf.gen.yaml` in one commit. No "remote-plus-local" hybrid mode.
- **No shims** preserving the old codepath for users who have already run `buf registry login`. The login becomes optional metadata, not a fallback.
- **No environment-variable toggles** for switching between local and remote plugins. One way to generate code: locally.

This rule is repeated in §13 (Definition of Done) and tested by acceptance criteria.

## 4. Problem Statement

### What's broken today

`packages/proto/buf.gen.yaml` declares **5 plugins, all `remote:` to `buf.build/...`**. Each `buf generate` invocation makes 5 HTTPS round-trips to BSR (one per plugin). Validation flows that exercise template generation (`vrooli scenario template validate`, scenario generation, `make generate`) hit this code path repeatedly. Anonymous BSR rate limits — undocumented but empirically a few hundred requests/hour — are easily exhausted in a single iteration session and recover slowly (failed retries extend the cooldown).

The two structural concerns:

1. **Concurrent agent scaling.** Vrooli's roadmap has many agents running simultaneously; an unauthenticated per-IP rate limit is an immediate ceiling on parallelism.
2. **Offline / sovereignty.** Vrooli's vision (local GPUs, eventual hardware appliances) makes "needs internet to generate code" an architectural smell. If a contributor's flight Wi-Fi drops, scenario codegen should not stop working.

### What login alone does not solve

`buf registry login` raises the rate limit ceiling but does not eliminate the network dependency. A logged-in user on a plane still cannot run `buf generate`. Login is therefore a **parallel** improvement (helps token-expiry hygiene, helps fetching new module versions) but is not the primary fix.

### Concurrent issue: token-expiry visibility

Buf personal tokens default to **1-month expiry** and silently fail authenticated calls when expired. There is no canonical `buf registry whoami` command in buf v1.37 — the only deterministic probe is reading `~/.netrc` for a `machine buf.build` entry. Operators have no in-product surface that tells them their token expired until codegen fails mid-task.

### Concurrent issue: onboarding pickup

`vrooli-onboarding` v2 acceptance criterion **OT-V2-FEATURE-COMPLETE** requires the wizard to cover *every* integration documented in `docs/configuration/integrations/`. BSR login is not currently documented there, so even after the v2 rework lands, operators of a fresh clone get no guidance about it. We need to add the per-integration documentation page **before** the v2 rework runs, so it gets picked up by the contract-driven wizard automatically.

## 5. Scope

### In scope

- Switch `packages/proto/buf.gen.yaml` from 5 `remote:` plugins to 5 `local:` plugins.
- Vendor the two BSR module dependencies (`googleapis/googleapis`, `bufbuild/protovalidate`) as `directory:` inputs so codegen is fully offline-capable.
- Add Go-based host-tool handlers for the codegen plugins: `protoc-gen-go`, `protoc-gen-es`, `protoc-gen-python` (`protoc-gen-pyi` is included with the same Python toolchain).
- Replace bash installer `scripts/migrate_candidates/tools/buf.sh` with a Go handler `internal/tools/buf/handler.go`.
- Promote `buf` to `.vrooli/service.json` `hostTools[]` with `required: true` (it is already mandatory for the proto pipeline; document the contract).
- Reference each new plugin tool in `.vrooli/service.json` `hostTools[]` so `vrooli setup` installs them.
- Add `docs/configuration/integrations/buf-bsr.md` declaring BSR login as an `external_sign_in_command` integration so onboarding's V2 wizard picks it up automatically.
- Add a probe command to vrooli's CLI surface — `vrooli auth status` — that reports BSR login state alongside other tracked sign-in tools (designed so future `claude /login`, `codex login`, `gh auth login` probes share the surface).
- Update `docs/configuration/host/tools.md` cross-references to point at the new plugin tools.
- Update the `react-vite` template's proto-related guidance (the `prototypes` work that hit the rate limit) to reference the local-plugin pipeline.
- Tests: handler invariants, manifest schema validation, codegen-output diff stability, vendored-module-freshness check.

### Out of scope

- Implementing the `integration-hub` scenario (deferred per `docs/configuration/integrations/README.md`). The buf-bsr connector page is added speculatively-but-deliberately because OT-V2-FEATURE-COMPLETE makes the docs the contract; the wizard step lights up when integration-hub ships.
- Migrating other scenarios' protobuf usages — there is one buf module (`packages/proto/`) and everything routes through it.
- Adding `buf publish` / module-push capability. We never publish to BSR.
- A bespoke "BSR login UI" in vrooli-onboarding pre-v2. The probe command is enough until the wizard catches up.
- Pinning Buf CLI to a specific minor version beyond what `internal/tools/buf/tool.json` already does (currently floats; we'll add a pin as part of this work).
- Supporting the `prompt:` style of declaring BSR creds at codegen time. We leave login fully optional once local plugins ship.

## 6. Current Technical Context

| Area | File / Component | Current state |
|---|---|---|
| buf module config | `packages/proto/buf.yaml` | v2 module, two BSR deps (`googleapis`, `protovalidate`), lint default w/ exceptions |
| Codegen config | `packages/proto/buf.gen.yaml` | 5 `remote:` plugins, 2 `module:` inputs, 1 `directory:` input |
| Generated code | `packages/proto/gen/{go,typescript,python}` | committed, validated by `make check` diff |
| Codegen entry | `packages/proto/Makefile` | `make generate`, `make lint`, `make breaking`, `make check` |
| buf binary install | `internal/tools/buf/tool.json` (manifest only, no handler) + `scripts/migrate_candidates/tools/buf.sh` (bash, downloads GitHub release v1.37.0) | Fragmented; bash script not registered with Vrooli's tool runtime |
| Host-tool registry | `internal/tools/<name>/tool.json` + optional `internal/tools/<name>/handler.go`, registered in `internal/runtime/registry.go` `customToolHandlers` map, drift-tested by `TestToolManifestsReferenceRegisteredHandlers` | Pattern is mature (`cloudflared`, `stripe`, `vault` all use it). Cross-platform: linux/darwin/windows |
| Project-level required tools | `.vrooli/service.json` `hostTools[]` | Has `git`, `curl`, `jq`, `yq`, `docker`, `go`, `node`. **Does not include `buf`.** |
| Validation flow that hit the rate limit | `internal/cli/scenariohandlers/template_runtime.go:813` (`validateRelocationProtoSources`) calls `buf lint` (not generate) but the rate-limit hit happened during full template iteration that also runs `buf generate` | `buf lint` does not contact BSR for plugins, only for module deps if cache-miss |
| Configuration substrate the wizard reads | `docs/configuration/` (README, host/, integrations/, secrets.md, scenarios.md, etc.) — **the wizard is contractually bound to implement what's documented here** (PRD OT-V2-FEATURE-COMPLETE) | Does not currently document buf BSR |
| Existing `external_sign_in_command` documentation | `docs/configuration/integrations/external-auth.md` lines 78-103 — exactly the pattern that fits buf, claude-code, codex, gh, cloudflared, stripe | Pattern is settled; needs a per-integration page |
| Onboarding wizard plan | `scenarios/vrooli-onboarding/docs/WIZARD_FLOW.md` Step 5 | Empty until integration-hub ships, but **picks up integration pages automatically** |

## 7. Target End State

### Codegen

- `cd packages/proto && make generate` succeeds with **zero outbound HTTPS requests** to `buf.build` after first install.
- `buf.gen.yaml`:
  ```yaml
  version: v2
  inputs:
    - directory: schemas
    - directory: vendor/googleapis
    - directory: vendor/protovalidate
  plugins:
    - local: protoc-gen-go
      out: gen/go
      opt: [paths=source_relative]
    - local: protoc-gen-es
      out: gen/typescript
      opt: [target=ts, import_extension=none]
    - local: protoc-gen-es
      out: gen/typescript/js
      opt: [target=js, import_extension=js]
    - local: protoc-gen-python
      out: gen/python
    - local: protoc-gen-pyi
      out: gen/python
  ```
- `packages/proto/vendor/{googleapis,protovalidate}/` contains the exported BSR modules used by Vrooli, refresh script documented.
- `packages/proto/buf.yaml` `deps:` is removed (the `directory:` inputs replace the BSR module fetches).

### Host tools

- `internal/tools/buf/` — `tool.json` (with `handler: "buf"`) + `handler.go` + `handler_test.go`. Cross-platform: downloads GitHub release binary on linux/darwin/windows.
- `internal/tools/protoc-gen-go/` — same shape; installs via `go install google.golang.org/protobuf/cmd/protoc-gen-go@<pinned>` using the `go` host tool already required.
- `internal/tools/protoc-gen-es/` — same shape; installs the npm package `@bufbuild/protoc-gen-es@<pinned>` to a vendored `node_modules/.bin/` location and adds it to `PATH` (via setup symlink), or downloads the published-binary build if upstream provides one.
- `internal/tools/protoc-gen-python/` — same shape; installs via `pip install --user protobuf==<pinned>` (which ships `protoc-gen-python` and `protoc-gen-pyi`).
- `internal/runtime/registry.go` `customToolHandlers` registers all four.
- `.vrooli/service.json` `hostTools[]` adds entries for all four with `required: true`, `when: ["setup", "develop"]`.
- `scripts/migrate_candidates/tools/buf.sh` is **deleted** (greenfield rule — see §3).

### BSR login surface

- `docs/configuration/integrations/buf-bsr.md` exists, declaring:
  - `auth.kind: external_sign_in_command`
  - `sign_in_command: buf registry login`
  - `probe: presence of "machine buf.build" line in $HOME/.netrc`
  - Token-expiry recommendation (see §8)
  - When login is needed (*only* for refreshing vendored module deps; never for codegen)
- `vrooli auth status` command exists in `cmd/vrooli/` (or the appropriate root CLI domain), reporting per-tool sign-in state, designed to be extended for `claude`, `codex`, `gh`, `cloudflared`. Output is human-friendly by default, `--json` available for scripting.
- The command is referenced from `docs/configuration/host/tools.md` and from `buf-bsr.md` so future onboarding picks it up.

### Onboarding alignment

- No code change in `scenarios/vrooli-onboarding/`. The integration page in `docs/configuration/integrations/buf-bsr.md` is enough — when V2 wizard runs, Step 5 reads the integrations folder and renders a card for buf-bsr automatically. (V2 ships separately; this plan only ensures the contract entry exists.)

### Documentation

- `docs/development/proto.md` (new) — codegen pipeline, why local plugins, how to refresh vendored modules, troubleshooting.
- `docs/configuration/host/tools.md` — cross-reference the new plugin tools.
- `templates/scenarios/react-vite/` — README/PRD reference updated to point at `docs/development/proto.md`.

## 8. Contract Decisions

### CD-1: Local plugins are the only plugin path

`remote:` plugins are forbidden in `packages/proto/buf.gen.yaml`. A test (§9 T-2) greps the file and fails CI on any `remote:` line. Future plugins added must follow the local-handler pattern.

### CD-2: BSR module deps are vendored, not fetched

Codegen reads from `packages/proto/vendor/googleapis/` and `packages/proto/vendor/protovalidate/` — the BSR `module:` input form is removed. A documented refresh script (`packages/proto/scripts/refresh-vendor.sh` or `make refresh-vendor`) regenerates the vendor tree for occasional bumps; this is the **one** code path that requires BSR access (and can run logged-in or anonymously).

### CD-3: Plugin versions are pinned and tracked centrally

Each plugin's pinned version lives in **two places**: the tool manifest's `version` field (e.g. `internal/tools/protoc-gen-go/tool.json#/version`) and the install command in the handler. Test T-3 asserts they match. Bumping a plugin = updating the manifest + handler in one PR; the `make check` proto-diff gate catches bit-level drift.

### CD-4: `buf registry login` is optional, not required

The `external_sign_in_command` integration page declares the probe but the wizard's UX language is "**Optional** — only needed if you plan to update vendored proto modules. Codegen works without it." This frames login as polish, not a blocker.

### CD-5: Token expiry policy — recommend "never" by default, 1 year for security-conscious teams

| Option | When operator picks | Recommendation |
|---|---|---|
| 1 month (default) | Never | Too short — silently fails after 30 days |
| 6 months | Rarely the right answer | Skip |
| 1 year | Security-conscious teams, shared/multi-operator hosts | **Recommended for teams** — annual rotation is reasonable housekeeping |
| **Never expires** | **Solo operator on a personal box with `chmod 600 ~/.netrc`** | **Recommended default** — read-only token, codegen never touches BSR after vendor refresh, expiry buys minimal security but causes silent failures months later |

Rationale: with local plugins (CD-1) and vendored modules (CD-2), the BSR token is consulted **only** during the rare `make refresh-vendor` operation. No publish scope is requested. A leaked read-only token's worst-case impact is reading public BSR modules already published by Vrooli — nothing the attacker couldn't do anonymously. Auto-expiry would, by contrast, silently fail a `buf export` months from now when the operator is mid-task.

The `buf-bsr.md` page documents this with the renewal procedure: `buf registry logout && buf registry login`. The `vrooli auth status` command reports token presence (via `.netrc` match) but **cannot** report expiry without making an authenticated test call (the token itself is opaque). To detect expiry the probe optionally runs `buf curl --schema buf.build/bufbuild/protovalidate -o /dev/null` (or equivalent) — if BSR returns 401/403, the token is expired/revoked. This call is gated behind `--check-expiry` to avoid making BSR requests on every status check.

### CD-6: Probe contract for `external_sign_in_command` is `.netrc` line presence

`buf registry whoami` does not exist in buf 1.37. The deterministic probe is:

```
healthy if: ~/.netrc contains a line matching ^machine\s+buf\.build\b
unknown otherwise
```

Optionally upgrade to `expired` when `--check-expiry` is passed and the test BSR call returns 401/403. This contract is documented in `buf-bsr.md` and implemented in `vrooli auth status`.

### CD-7: `vrooli auth status` is the single surface for sign-in state

Today it reports `buf` only. Future-extension contract:

```go
type SignInProbe interface {
    Name() string                    // "buf", "claude", "codex", ...
    Probe(ctx context.Context) ProbeResult
}

type ProbeResult struct {
    State    SignInState   // "signed_in", "signed_out", "expired", "unknown"
    Detail   string        // e.g. "token in ~/.netrc; expiry not checked"
    SignInCommand []string // e.g. ["buf", "registry", "login"]
}
```

Per `cli-steer`: human-default output, `--json` available, built on `cli-core`'s `cliapp` scaffolding. The command lives in `internal/cli/<...>/` Go-only — no bash. Lives at the **vrooli root CLI** (not a scenario CLI) because sign-in state is a host-level concern.

### CD-8: Plugin install handlers are pure Go

No shelling out to bash/.sh files. Cross-platform via `runtime.GOOS` / `runtime.GOARCH` switches inside the handler (mirroring `cloudflared/handler.go` and `stripe/handler.go`). The `protoc-gen-es` Node-based plugin is the awkward case — install via `npm install --prefix <vendor-dir> @bufbuild/protoc-gen-es@<pinned>` (Node is already a required host tool) and symlink the `protoc-gen-es` shim into `~/.local/bin/`. Documented as a contract decision because it is the one departure from "ship a single binary" elsewhere in the registry.

## 9. Implementation Strategy (Phased)

Phases are ordered for safe interruption — each phase leaves the tree green and tests passing.

### Phase A: Vendor BSR module dependencies (1 commit)

1. `cd packages/proto && buf export buf.build/googleapis/googleapis -o vendor/googleapis` (one-time, requires internet, can be done while logged in).
2. `buf export buf.build/bufbuild/protovalidate -o vendor/protovalidate`.
3. Add `packages/proto/scripts/refresh-vendor.sh` (Go would be nicer but a 30-line shell helper is acceptable for this single tooling script — alternative: tiny `cmd/refresh-proto-vendor/main.go`; pick Go to honor the cross-platform constraint).
4. Switch `inputs:` in `buf.gen.yaml` to `directory: vendor/googleapis` and `directory: vendor/protovalidate`.
5. Remove `deps:` from `buf.yaml`.
6. Run `make generate` — output should be byte-identical to before.
7. Run `make check` — should be green.

**Stop / acceptance:** `git diff packages/proto/gen/` is empty after `make generate`.

### Phase B: Add plugin host-tool handlers (4 commits, parallel-safe)

Each in its own commit because each is independent and reviewable.

1. **Phase B.1 — `protoc-gen-go`**:
   - Add `internal/tools/protoc-gen-go/tool.json` (`handler: "protoc_gen_go"`, `version: "v1.36.x"`, `commands: ["protoc-gen-go"]`, `versionArgs: ["--version"]`).
   - Add `internal/tools/protoc-gen-go/handler.go` with `Inspect`/`Apply` mirroring `cloudflared/handler.go` shape; install via `go install google.golang.org/protobuf/cmd/protoc-gen-go@<version>` with `GOBIN` set to a registry-known location (e.g. `$XDG_BIN_HOME` falling back to `~/.local/bin`).
   - Add `handler_test.go` covering: already-installed shortcut, install on clean linux, install on darwin, unsupported windows path (or supported via same `go install` since Go is cross-platform — preferred).
   - Register in `internal/runtime/registry.go` `customToolHandlers["protoc_gen_go"] = protocgengo.NewHandler`.
   - Add to `.vrooli/service.json` `hostTools[]` with `required: true, when: ["setup", "develop"], environments: ["development", "production"]`.
2. **Phase B.2 — `protoc-gen-es`** (Node-based; see CD-8):
   - Vendored `node_modules/` strategy: install into a project-local well-known dir (e.g. `~/.cache/vrooli/protoc-plugins/node`) and symlink the executable.
   - Pin via package.json or `npm install --prefix <dir> @bufbuild/protoc-gen-es@<pinned>`.
3. **Phase B.3 — `protoc-gen-python` / `protoc-gen-pyi`**:
   - Single handler covering both binaries (they ship together with the `protobuf` pip package).
   - Install via `pip install --user protobuf==<pinned>` or, preferred, set up a venv at `~/.cache/vrooli/protoc-plugins/python` and install there + symlink shims.
4. **Phase B.4 — `buf` handler upgrade**:
   - Add `internal/tools/buf/handler.go` (currently only `tool.json` exists).
   - Replace shell installer with Go handler that downloads `buf-<OS>-<ARCH>` from GitHub releases (matches `scripts/migrate_candidates/tools/buf.sh` behavior) — pin to v1.37.0 in `tool.json#/version`.
   - Update `internal/tools/buf/tool.json` to set `handler: "buf"` and `version: "v1.37.0"`.
   - Promote `buf` to `.vrooli/service.json` `hostTools[]` with `required: true`.
   - **Delete** `scripts/migrate_candidates/tools/buf.sh` (greenfield rule).
   - Add `handler_test.go`.

**Stop / acceptance after each phase:**
- `go test ./internal/runtime/...` passes (manifest-handler invariant)
- `go test ./internal/tools/<name>/...` passes
- `go test ./internal/setup/...` passes
- A clean `make setup` on a Linux VM (or container) successfully installs the new tool.

### Phase C: Switch `buf.gen.yaml` to local plugins (1 commit)

1. Replace each `remote:` plugin entry with the corresponding `local:` form.
2. Run `make generate` — output should be byte-identical to before (sanity check).
3. Run `make check` to confirm.
4. Add a guard test under `packages/proto/scripts_test.go` (or appropriate test location) that asserts `buf.gen.yaml` contains zero `remote:` lines (CD-1).

**Stop / acceptance:** `make check` green, no BSR network traffic during `make generate` (verify with `strace -f -e trace=connect` or `tcpdump`-equivalent — see T-7).

### Phase D: BSR login documentation surface (1 commit)

1. Add `docs/configuration/integrations/buf-bsr.md` following the per-integration page pattern in `docs/configuration/integrations/README.md`. Page sections (mirroring `video-providers.md`):
   - Why this page exists (BSR is optional now; here's when you'd want login).
   - Auth pattern (`external_sign_in_command`).
   - Sign-in command, probe contract, expiry policy (CD-5), renewal procedure.
   - Wizard pickup notes (V2 wizard surfaces this automatically when integration-hub ships).
2. Cross-link from `docs/configuration/integrations/external-auth.md` "Examples" line and from `docs/configuration/integrations/README.md` "Existing per-integration pages" list.
3. Cross-link from `docs/configuration/host/tools.md` (proto plugins reference buf-bsr for context).

**Stop / acceptance:** `docs/configuration/integrations/buf-bsr.md` exists; lint passes (`lychee` for link-checking already in repo).

### Phase E: `vrooli auth status` command (1 commit)

1. Decide command location — most likely a new domain in the root vrooli CLI (`internal/cli/authcli/` or similar) since this is a host-level concern, not scenario-specific. Per `cli-steer` Section 4, follow the standard structure (`domains/auth/{register,status,output}.go`).
2. Implement `vrooli auth status` with the `SignInProbe` interface (CD-7). Initial probes: `buf` only.
3. Add `--json` flag (cli-core's `cliutil.JSONFlag`).
4. Add `--check-expiry` flag (CD-5; gated to avoid BSR traffic on default invocations).
5. Test coverage: probe-state matrix (signed_in, signed_out, expired-when-flag-set, unknown).
6. Register the command in the root CLI.

**Stop / acceptance:** `vrooli auth status` runs on a fresh box and reports `buf: signed_out` with the documented sign-in command; after running `buf registry login`, reports `buf: signed_in`.

### Phase F: Developer documentation (1 commit)

1. Add `docs/development/proto.md` covering:
   - Pipeline architecture (5 local plugins, 2 vendored modules, no BSR codegen path).
   - How to add a new plugin (mirror existing handler).
   - How to refresh vendored modules.
   - How to bump a plugin version (manifest + handler in one PR, `make check` validates).
   - Troubleshooting matrix (drift errors, clean-rebuild instructions).
2. Update `templates/scenarios/react-vite/` proto-related references to point here.
3. Add a brief mention in repo-root `docs/README.md` index.

**Stop / acceptance:** `docs/development/proto.md` linked from at least 2 other docs; lychee link-check green.

## 10. Testing Plan

| ID | Test | Where | What it asserts |
|---|---|---|---|
| T-1 | Manifest-handler invariant | `internal/runtime/registry_test.go` (existing `TestToolManifestsReferenceRegisteredHandlers`) | New tool.json files reference handlers registered in `customToolHandlers` |
| T-2 | No `remote:` plugins in `buf.gen.yaml` | `packages/proto/scripts_test.go` (new) — Go test that reads `buf.gen.yaml` and fails if any line matches `^\s*-\s*remote:` | Enforces CD-1 in CI |
| T-3 | Plugin pin consistency | `internal/tools/<plugin>/handler_test.go` | Asserts manifest `version` field matches the version string baked into the install command |
| T-4 | Codegen byte-stability | `packages/proto/Makefile` `check` target (already exists) | `make generate` produces no diff vs committed `gen/` — runs in CI |
| T-5 | Handler install paths | `internal/tools/<plugin>/handler_test.go` | Linux apt path, darwin brew path, windows winget/`go install` path each install correctly (use httptest fixture for download steps) |
| T-6 | `vrooli auth status` probe states | `internal/cli/authcli/status_test.go` (new) | Each `SignInProbe` reports correct state for {missing .netrc, present .netrc, expired token (with --check-expiry)} |
| T-7 | Codegen offline acceptance | `packages/proto/scripts_test.go` (new) — runs `make generate` with `HTTPS_PROXY=http://127.0.0.1:9` (deliberately unreachable) and asserts success | Codegen does not require network |
| T-8 | Setup installs plugins idempotently | `internal/setup/setup_test.go` (existing harness) — extend with new tools | Re-running `vrooli setup` after plugins installed is a no-op |
| T-9 | `vrooli scenario template validate` works without BSR | Existing `template_runtime_test.go` — extend with a fault-injection variant where BSR is unreachable | `buf lint` path no longer hits BSR (because vendored modules + local plugins) |
| T-10 | Documented integration is parseable | `docs/configuration/integrations/buf-bsr.md` — add to existing markdown lint / link-check pass; future: schema-validate against a connector schema once integration-hub ships | Page format matches the contract template |

All tests run in `make test` and CI's existing pipeline. No manual checklists — per the repo's testing convention.

## 11. Rollout / Validation Checklist

Each item is automated where possible.

- [ ] `make setup` on a clean Linux VM installs all 5 host tools (`buf`, `protoc-gen-go`, `protoc-gen-es`, `protoc-gen-python`, `protoc-gen-pyi`) without internet beyond the initial fetches.
- [ ] `make setup` on macOS (manual, or matrix CI if present) installs all 5 tools.
- [ ] `make setup` on Windows (best-effort; document if matrix CI absent) installs the 4 cross-platform tools (`protoc-gen-es` Node path, others via `go install` / `pip`).
- [ ] `cd packages/proto && make generate` produces zero diff under `gen/`.
- [ ] `cd packages/proto && make generate` succeeds with `HTTPS_PROXY=http://127.0.0.1:9` (proves no BSR contact).
- [ ] `vrooli scenario template validate <fixture-with-protos>` succeeds end-to-end without rate-limiting (run 50× in a tight loop as a regression check).
- [ ] `vrooli auth status` returns `buf: signed_out` on a fresh `.netrc`-less account.
- [ ] After `buf registry login` with a 1-year token, `vrooli auth status` returns `buf: signed_in`.
- [ ] `vrooli auth status --check-expiry` with an expired token reports `buf: expired`.
- [ ] `docs/configuration/integrations/buf-bsr.md` is linked from `external-auth.md` and `integrations/README.md`.
- [ ] `docs/development/proto.md` is linked from `docs/README.md`.
- [ ] `scripts/migrate_candidates/tools/buf.sh` is **deleted**.
- [ ] `git grep "remote:" packages/proto/buf.gen.yaml` returns no matches.

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `protoc-gen-es` produces different bytes than the BSR remote variant | Low | High (CI diff fails on first run) | Phase A vendoring is a no-op on inputs; Phase C plugin switch is the bit-stable change. Run `make check` after Phase C. If diff appears, pin to the exact remote-plugin version Buf has been running. |
| Python plugin name discoverability — `protoc-gen-python` may not be on `PATH` after `pip install` if the user has multiple Python installs | Medium | Medium | Handler installs into a Vrooli-owned venv at `~/.cache/vrooli/protoc-plugins/python` and symlinks `protoc-gen-python` and `protoc-gen-pyi` shims to `~/.local/bin/`. Test the symlink works on linux/darwin. |
| Node-based `protoc-gen-es` shim doesn't survive PATH changes | Medium | Medium | Symlink lives in `~/.local/bin/` (already on `PATH` per Vrooli setup conventions), pointing at `~/.cache/vrooli/protoc-plugins/node/node_modules/.bin/protoc-gen-es`. Probe with `which protoc-gen-es` in the handler's `Inspect`. |
| Vendored module drift from upstream | Medium | Low (only matters when consuming code uses new fields) | `packages/proto/scripts/refresh-vendor.sh` is documented; bump cadence is "when needed", not scheduled. CI check: `make check` post-refresh must be green or the bump is rolled back. |
| Plugin version drift between contributors | Low | Medium | T-3 (manifest pin matches handler-baked pin). `make check` catches generated-code drift in CI. New contributors see correct versions installed by `make setup`. |
| BSR rate-limit still hits during `buf export` for vendor refresh | Low | Low | Refresh is rare (months between bumps); if it hits, operator runs `buf registry login` per CD-5. Document this as the *one* time login matters. |
| Test fixture for offline codegen check (T-7) is flaky | Low | Low | `HTTPS_PROXY=http://127.0.0.1:9` is deterministic; no network access possible at that target. Skip on Windows where `HTTPS_PROXY` semantics differ. |
| `vrooli auth status` collides with future scenario CLIs that have their own `auth status` | Low | Low | Per `cli-steer`, sign-in state is a *host* concern; it lives at root CLI. Scenario CLIs that need their own auth do so under their own subcommands. Document precedence in `cli-steer` once a real conflict appears. |
| Operator opens `~/.netrc` permissions question (BSR token is plaintext) | Low | Medium | `buf-bsr.md` notes the `.netrc` storage and references the standard `chmod 600 ~/.netrc` advice. Same posture as `gh auth login` and other tools. |
| `protoc-gen-es` Node toolchain deemed too heavy for a "production" environment profile | Low | Low | TS/JS codegen is required for the scenarios that use protos. If a future "minimal" profile wants to skip it, the host-tool entry's `environments:` field already supports it. Default: `["development"]` for the Node path; require Node anyway. |

## 13. Non-goals / Prohibited Patterns

Per §3 (greenfield rule):

- **No fallback to `remote:` plugins.** Not via env var, not via `BUF_REMOTE=1`, not via "if local install fails, fall back". If local install fails, fix the install path.
- **No bash installer scripts** for the new plugins or for buf. All install logic is Go in `internal/tools/<name>/handler.go`.
- **No "sign in to BSR or your build won't work" framing** anywhere in user-facing docs. Login is *only* needed for vendor refresh; codegen is unconditional.
- **No bespoke BSR sign-in UI** in vrooli-onboarding pre-V2. The contract page in `docs/configuration/integrations/buf-bsr.md` is enough; the V2 wizard rework picks it up. Adding a one-off wizard step now would be thrown away.
- **No `buf` invocations from scenario code or skill prompts.** Scenarios never run `buf` directly — codegen is a packages/proto concern. (Memory: `feedback_skills_use_cli_never_api`.)
- **No new CLI surface for buf-specific operations.** `vrooli auth status` is generic, host-level, and extends to other tools. There is no `vrooli buf login` or similar.
- **No commits with stale `gen/` output.** `make check` in CI rejects.

## 14. Definition of Done

All of the following must be objectively true:

1. `packages/proto/buf.gen.yaml` contains **zero** `remote:` plugin entries (T-2 enforces).
2. `make generate` produces byte-identical output to the pre-change `gen/` tree (Phase A acceptance + T-4).
3. `make generate` succeeds with `HTTPS_PROXY=http://127.0.0.1:9` (T-7).
4. `make setup` on a clean Linux VM installs all 5 host tools (`buf`, `protoc-gen-go`, `protoc-gen-es`, `protoc-gen-python`, `protoc-gen-pyi`) and `make generate` works immediately afterward.
5. `internal/runtime/registry.go` `customToolHandlers` includes `buf`, `protoc_gen_go`, `protoc_gen_es`, `protoc_gen_python` (T-1 passes).
6. `scripts/migrate_candidates/tools/buf.sh` is deleted from the working tree.
7. `.vrooli/service.json` `hostTools[]` lists `buf` and the four plugin tools as `required: true`.
8. `docs/configuration/integrations/buf-bsr.md` exists, follows the established per-integration page format, and is linked from `integrations/README.md` and `external-auth.md`.
9. `docs/development/proto.md` exists and is linked from `docs/README.md`.
10. `vrooli auth status` reports BSR login state correctly across {signed_in, signed_out, expired-with-flag} states.
11. All new tests (T-1 through T-10) pass in CI; existing tests remain green.
12. No file in this PR's diff contains a `remote:`-to-BSR plugin reference, a fallback-to-bash install path, or a "log in or codegen breaks" doc statement.
13. PR description includes the `vrooli scenario template validate` 50×-loop result demonstrating no rate limit hits.

---

## Appendix A: Recommended action for "log in *now*"

The user has begun `buf registry login`. Recommendation, separable from this plan's execution:

1. Pick **"never expires"** as the token lifetime (CD-5) — solo operator on a personal box, read-only scope, no value lost to expiry. Pick **1 year** instead if multiple operators share this host or the box is exposed beyond personal use.
2. Paste the token to complete the login. This unblocks immediate iteration on whatever proto work was in flight.
3. `chmod 600 ~/.netrc` once login completes (idempotent; do not assume default file mode is restrictive enough).
4. Once Phase A–C of this plan land, the login becomes purely-optional metadata used only for `make refresh-vendor`.

## Appendix B: Why the agent's "buf registry login" advice was right but incomplete

The agent's analysis (paste-quoted in the user's question) correctly diagnosed BSR remote-plugin rate-limiting and identified `buf registry login` as the path-of-least-resistance. The two gaps it should have flagged:

1. Login *raises the ceiling but doesn't remove the network dependency*. Local-plugin switch does both.
2. Buf's documented self-hosted BSR is enterprise-licensed and not a viable "set up a local version" option. The realistic "local version" is local plugins + vendored modules (this plan).

Both gaps are now addressed.
