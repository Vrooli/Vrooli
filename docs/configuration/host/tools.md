# Host Tools

Host tools are command-line dependencies Vrooli expects to find on the host (or installs through its own runtime if missing) — `git`, `curl`, `jq`, `docker`, `cloudflared`, `vault`, etc. They are declared per-resource and per-scenario, registered globally at the project level, and opted into by the operator.

## What lives where

| Concern | File | Field |
|---|---|---|
| What this tool is, how to install, how to verify | `internal/tools/<name>/tool.json` | top-level manifest |
| Custom Go install handler (when standard package install isn't enough) | `internal/tools/<name>/*.go` (registered in `internal/runtime/registry.go`) | `customToolHandlers` map |
| Top-level project requirements | `.vrooli/service.json` | `hostTools[]` (each entry: `hostRequirement`) |
| Per-scenario requirements | `scenarios/<name>/.vrooli/service.json` | `hostTools[]` |
| Per-resource requirements | `resources/<name>/resource.json` | `hostTools[]` |
| Operator opt-in for non-required tools | `.vrooli/operator-state.json` | `host_tools.<name>.opted_in` |

## How tools are discovered

There is one canonical registry, drift-protected by `internal/runtime/manifests_test.go`:

- The filesystem at `internal/tools/<name>/tool.json` is the list of known tools.
- The Go map `customToolHandlers` in `internal/runtime/registry.go` lists tools that need custom install logic (e.g. `cloudflared`, `stripe`, `vault`).
- The invariant test `TestToolManifestsReferenceRegisteredHandlers` ensures every manifest with a custom `handler` field has a corresponding registered Go handler. No drift between code and config.

## Deployment classification

Every tool manifest has a `bundling` classification. The deployment contract is
the single definition of the vocabulary: see
[`deployment-contract.md`](../../resources/deployment-contract.md#deployment-eligibility-axes).

- `vendorable` tools have a checksummed, per-platform artifact that a desktop
  bundle can stage.
- `host-required` tools must be present on the target host and are never
  silently assumed to be bundled. Display infrastructure and `secret-tool` are
  examples.
- `prohibited` tools must not enter a desktop bundle.

`privilege` is a separate axis. It describes the maximum permission needed to
install or operate the tool on a Vrooli-owned host; it does **not** say whether
the tool may ship in a desktop application. For ordinary tools it is derived
from the install mechanism per platform. A manifest only declares it explicitly
when that derivation would be wrong, and then supplies `privilegeReason`.
`elevated` work remains confined to Vrooli's explicit project setup boundary.

## Linux credential storage

`secret-tool` is the Linux-only `libsecret` command client used by the
managed-resource secure-store seam. Installing it is necessary but not enough:
a shared managed resource first performs a non-secret store/read/delete probe
against the active user Secret Service. If that probe fails, shared bootstrap
stops before initialization; Vrooli never falls back to a plaintext state file.

### Repairing the Linux credential-store prerequisite

For a fresh setup with Vault enabled, `vrooli setup` installs this required
tool during its single privileged setup pass. To repair an existing host, use
the Vrooli host-tool installer. The install can require an interactive
administrator authentication prompt because `libsecret-tools` is a system
package. Do not substitute a raw package-manager command.

```bash
vrooli host install secret-tool --sudo-mode=ask
command -v secret-tool
vrooli resource start vault
vrooli resource status vault
```

The Vault start performs the non-secret Secret Service probe. If it fails,
start and unlock the active user's Secret Service session, then rerun the
`vrooli resource start vault` command. Do not initialize a shared managed
resource until that probe passes.

Onboarding consumes the filesystem registry; it does not maintain its own list.

## hostRequirement shape

Each entry in a `hostTools[]` array is a `hostRequirement`:

```json
{
  "name": "cloudflared",
  "required": false,
  "reason": "Cloudflare tunnel client for app-monitor and remote access",
  "when": ["develop"],
  "environments": ["development"],
  "platforms": ["linux", "macos"]
}
```

| Field | Required | Purpose |
|---|---|---|
| `name` | yes | Tool identifier matching the registry name |
| `required` | yes | Whether the install is mandatory (true) or operator-opt-in (false) |
| `reason` | yes | Human-readable justification, shown to operators during install |
| `when` | no | Phases the tool is needed in (`setup`, `develop`, etc.) |
| `environments` | no | Environment profiles this applies to (`development`, `production`, `minimal`) |
| `platforms` | no | OSes the tool applies to. Omit for all |
| `manual` | no | If true, Vrooli does not auto-install — operator handles it |
| `notes` | no | Freeform internal annotation |

## Opt-in flow

For `required: true` tools: installed automatically as part of `vrooli setup`. No operator choice; the tool is mandatory for the manifest's claimed function.

For `required: false` tools: presented to the operator at the host step of onboarding, with `reason` shown. Operator opts in via the wizard, which writes to `operator-state.json`:

```json
"host_tools": {
  "cloudflared": { "opted_in": true }
}
```

A tool absent from `host_tools.*` falls back to its manifest `required` field — required tools install regardless; non-required tools default to "not opted in."

## Risk indicators

Tools do not currently have a `risk` field on their manifest (unlike safeguards — see [`safeguards.md`](safeguards.md)). Most tools are package installs with low risk. `privilege` is not a replacement for risk: it is a machine-readable deployment and setup gate, while risk is an operator-facing assessment of host-state impact. If a tool's installation has meaningful side effects (root privilege escalation, modifying system services, etc.), document it in `notes` and consider whether it should be a safeguard rather than a tool.

## Adding a new tool

1. Create `internal/tools/<name>/tool.json` conforming to [`tool.schema.json`](../../../.vrooli/schemas/tool.schema.json).
2. If the tool needs custom install logic (not just a package install), add a Go handler under `path:internal/tools/<name>/`, register it in `internal/runtime/registry.go` `customToolHandlers`, and reference the handler name in the manifest.
3. Reference the tool from the consuming `service.json` or `resource.json` `hostTools[]` array using a `hostRequirement` entry.
4. Verify with `go test ./internal/runtime/...` that the manifest-vs-handler invariant passes.

The wizard surfaces the new tool automatically once the manifest and any consumer references are in place.

## Versioned host tools and toolchain authority

`internal/tools/<name>/tool.json` is the single authority for a host tool's
exact release when it declares `version` plus checksum-verified `source`
targets. The generic host-tool handler treats a different detected version as
not ready and repairs it from the declared release target; it never treats a
similarly named OS package as equivalent.

Directory releases may also declare `runtimeEnv`: environment-variable names
mapped to paths inside the extracted tool directory. The generated launcher
resolves and exports those paths, so a host's ambient runtime configuration
cannot redirect a managed tool to an unrelated installation.

Go is the workspace example because it is both a host tool and the compiler.
Its manifest owns the exact patch release, `go.mod`'s `toolchain` directive
requests that same release for Go-native development, and CI extracts the
manifest version before calling `actions/setup-go`. The `go` directive is kept
separate: it is the module compatibility floor, not the installed compiler
release. The contract test in `internal/runtime/go_toolchain_contract_test.go`
guards these consumers against drift.

## Sign-in state for host tools

Some tools are runtime-authenticated rather than just installed — the operator runs a sign-in command (`buf registry login`, `claude /login`, future: `codex login`, `gh auth login`, ...) and the tool stores its own credentials in its own config dir. These follow the [`external_sign_in_command`](../integrations/external-auth.md#external_sign_in_command) integration pattern, with a per-tool integration page under [`path:docs/configuration/integrations/`](../integrations/README.md) (e.g. [`buf-bsr.md`](../integrations/buf-bsr.md)).

The host-level surface for inspecting sign-in state is the **`vrooli auth status`** command. It runs each registered probe in name order and reports `signed_in` / `signed_out` / `expired` / `unknown`. Default invocations are offline; `--check-expiry` enables an authenticated upstream call to distinguish a present-but-stale token from a healthy one.

```bash
vrooli auth status                   # human-readable table
vrooli auth status --json            # JSON for scripting
vrooli auth status --check-expiry    # additionally validate against upstream
```

New tools register a probe by implementing `auth.SignInProbe` in `path:internal/app/auth/` and adding it to `auth.DefaultProbes()`. The CLI surface stays fixed; only the probe set grows.

## See also

- [`safeguards.md`](safeguards.md) — host-state-modifying counterparts to tools
- [`../architecture.md#resolution-order`](../architecture.md#resolution-order) — opt-in resolution rules
- [`../integrations/buf-bsr.md`](../integrations/buf-bsr.md) — Buf Schema Registry sign-in (worked `external_sign_in_command` example)
- [`../../development/proto.md`](../../development/proto.md) — proto codegen pipeline (consumes `buf` and the proto plugins)
- [`internal/runtime/registry.go`](../../../internal/runtime/registry.go) — the canonical handler map
