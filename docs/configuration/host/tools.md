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
| Operator opt-in for non-required tools | `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` | `host_tools.<name>.opted_in` |

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

The Linux package-manager probe recognizes `apt-get`, `dnf`, `yum`, `zypper`,
`pacman`, `apk`, and `brew` in that order. A release-backed tool that has no
upstream `linux/arm64` artifact must declare a reason at
`source.unsupported["linux/arm64"]`; this prevents a missing target from being
mistaken for an installable route. `sd`, `sd-gpu`, and
`realesrgan-ncnn-vulkan` use that explicit unsupported declaration. The
manifest invariant test `TestReleaseManifestsDeclareLinuxArm64RouteOrUnsupportedReason`
guards the contract.

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

## Platform support ladder

Vrooli reports platform support in two separate rungs:

| Rung | Linux | macOS | Windows |
|---|---|---|---|
| Host provisioning (`vrooli setup`) | Supported | Build-verified | Build-verified through winget, Chocolatey, or Scoop when one is available |
| Scenario runtime (`vrooli develop`, resource and scenario lifecycle) | Supported | Supported | Not supported; lifecycle steps execute through the shell-defined `bash` contract |

Windows support therefore means that Vrooli can inspect and provision host tools
through the detected package manager. It does not claim that scenario lifecycle
processes or service management run on Windows. When no Windows package manager
is present, setup reports that capability as unavailable instead of assuming
winget or attempting a bootstrap.

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

## Coding-agent updates

Full deployments invoke the control plane through `cliinvoke`. Standalone Web Console desktop builds embed `packages/harness-lifecycle`: they acquire native artifacts declared by each harness's `native_artifacts` data, verify the upstream registry SHA-512 or release-manifest digest, and write receipts in user-owned version prefixes. They do not call npm, Node, the Dependency Analyzer, or the `vrooli` wrapper. Grok's native artifact is decoded with the compiled portable Brotli reader, with cancellation and a 512 MiB output bound before version and receipt verification.

Manage offers **Check installation** and **Review managed copies to remove**. Removal sends the preview identity back to the owner; a changed set requires a new preview. Found copies are listed as preserved and never renamed or removed. The CLI equivalent is `resource update <name> --action preview-cleanup`, then `--action uninstall --cleanup-plan <id>`; `--action doctor` is read-only.


### Harness state

Keep three facts separate when describing a coding-agent harness:

| Fact | Meaning | Authority |
|---|---|---|
| Found copy | A harness executable is discoverable on this host. Discovery does not make the copy managed. | Host discovery |
| Managed copy | A copy is installed and maintained through Vrooli's managed lifecycle. A found system or user copy can remain unmanaged. | Control-plane lifecycle |
| Managed verbs | The lifecycle actions supported for this harness, such as install, update, repair, rollback, or remove. Support varies by harness and installation method. | Harness descriptor and lifecycle implementation |

The update states below describe the update observation and available update action. They do not replace the found/managed distinction or imply that every harness supports every managed verb. The target implementation records these facts separately; do not infer management from discovery alone.

`vrooli resource update <name>` updates a coding-agent resource and repairs how it
is installed. It is the only owner of that work; Web Console and other scenarios
invoke it. `<name>` is `claude-code`, `codex`, `opencode`, `grok` or `antigravity`.

| Command | Effect |
|---|---|
| `vrooli resource update <name>` | Install the latest release and select it. Never downgrades. |
| `vrooli resource update <name> --check`, or `vrooli resource update-check <name>` | Report state and checks. Changes nothing. Bridge admits `update-check` with read scope. |
| `... --action use-managed-copy` | Make Vrooli's copy run first. |
| `... --action link-managed-copy` | Replace a standalone file at `~/.local/bin/<binary>` with the link. |
| `... --action rollback` | Link the previous version directory. |

With `--json`, the result is under `harness_update`, with the observation in
`report`. A failed action exits 1 and still prints the result.

### States

| State | Meaning | Primary action |
|---|---|---|
| `missing` | No copy on `PATH`. | `install` |
| `shadowed` | Another copy runs first. The report names what runs and what is managed. | `use_managed_copy`, or none for a system package |
| `unlinked` | `~/.local/bin/<binary>` is a standalone file or a foreign link. | `link_managed_copy` |
| `needs_setup` | A file an action must change belongs to root. | None. `operator_command` is `sudo vrooli setup --include-optional`. |
| `update_ready` | Vrooli's copy runs, it is behind, and every check passed. | `update` |
| `up_to_date` | Vrooli's copy runs and it is current. | None |
| `check_failed` | Something could not be verified, or something would fail the update. The reason says which. | `retry` |

`update_ready` means the update can succeed. A denied package, an unreachable
Dependency Analyzer, or a missing Node.js makes it `check_failed`.

### What each repair does

- **`use-managed-copy`** renames the shadowing file to
  `<file>.vrooli-replaced-<UTC time>` beside itself. Shell startup files are never
  edited, so the `PATH` line stays and resolution falls through to the Vrooli shim
  and the managed link. To undo it, rename the file back. An upstream self-update
  can recreate the file; the next check reports `shadowed` again.
- **`link-managed-copy`** renames the standalone file the same way and links the
  newest managed version. If that version is older than what runs, it installs the
  latest instead. A failure restores the file.
- **`rollback`** verifies the previous version directory against its receipt and
  relinks it. It installs nothing. `update` still refuses to downgrade.
- Replaced files in `PATH` stay beside themselves. Nothing here elevates.

A file in `~/.opencode/bin` counts as running first when a shell startup file puts
that directory on `PATH`, even when the caller's own `PATH` orders it later: that
is what a session's shell runs.

### Version sources

The latest version is what an update can install: the npm registry for npm-acquired
agents, and the per-platform release manifest for Antigravity, whose archive is
verified against the manifest's SHA-512. Antigravity is not an npm package, so it
does not pass through the Dependency Analyzer.

npm approvals are exact, so each new release starts `out_of_range`. `update` then
moves the existing record to that exact version through `deps approved upsert`,
keeping its notes and scope. A `denied`, `deprecated` or scenario-restricted record
blocks the update.

The record says what happened. `approvedBy` names the actor and `rationale` reads
"harness update to X by <actor>". `lastReviewed` stays and `approvedDate` is cleared,
because an update is not a dependency review. The actor is the agent runtime and
session when an agent signal is present, even if the caller claims otherwise. Web
Console passes its verified operator with `--on-behalf-of`. This is attribution, not
authentication. Agents may run this verb; agent identity alone is not a reason to
refuse maintenance. `--action refresh-catalog` names the actor in the same way, in the
policy's provenance.

If anything fails after the record moves, the previous record goes back. `rollback`
also restores it from `previous-approval.json` in the version directory, unless a
later decision changed the record. A gateway outage does not block a rollback; the
message says the approval still names the newer version.

**Antigravity's trust decision.** It rests on TLS to the manifest service. That one
service supplies the download location and the checksum, so the SHA-512 detects
corruption, not a compromised service, and a compromised service gets code execution
when the candidate's `--version` runs. Accepted because this is the channel upstream's
own installer trusts, and Vrooli adds these limits: the download and every redirect
must stay on the manifest host or under
`https://storage.googleapis.com/antigravity-public/`; the archive is capped at 1 GiB
and the executable at 2 GiB; and only a regular tar entry can become the executable.
Updating past the `resource.json` pin is not reviewed by the Dependency Analyzer.

`upstream_cli.version_pinned` is the version the permission policy was reviewed
against. It never selects what installs. `newer_than_reviewed` compares it with the
running version. An agent without that field uses its acquisition version.

Grok's npm entry point runs `~/.grok/bin/grok` when that file exists and only
otherwise unpacks the package's binary there. Installs disable lifecycle scripts, so
an older file there outlives every package update, and the package reports the older
version. The check reports this as `shadowed`. Updates, repairs and rollback retire
the stale entry point and let the launcher rebuild it. Older `grok-<version>` files
stay. A regular file at that path is kept until the rebuilt one verifies, and then
removed; if the action fails, the old entry point goes back.

### Effect on running sessions

The update swaps a symlink and leaves the previous version directory in place, so a
running session keeps its version. Every launch resolves the binary again, and a
resumed session is a new process, so new and resumed sessions use the new version.

## See also

- [`safeguards.md`](safeguards.md) — host-state-modifying counterparts to tools
- [`../architecture.md#resolution-order`](../architecture.md#resolution-order) — opt-in resolution rules
- [`../integrations/buf-bsr.md`](../integrations/buf-bsr.md) — Buf Schema Registry sign-in (worked `external_sign_in_command` example)
- [`../../development/proto.md`](../../development/proto.md) — proto codegen pipeline (consumes `buf` and the proto plugins)
- [`internal/runtime/registry.go`](../../../internal/runtime/registry.go) — the canonical handler map
