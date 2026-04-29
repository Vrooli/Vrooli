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

Tools do not currently have a `risk` field on their manifest (unlike safeguards — see [`safeguards.md`](safeguards.md)). Most tools are package installs with low risk. If a tool's installation has meaningful side effects (root privilege escalation, modifying system services, etc.), document it in `notes` and consider whether it should be a safeguard rather than a tool.

## Adding a new tool

1. Create `internal/tools/<name>/tool.json` conforming to [`tool.schema.json`](../../../.vrooli/schemas/tool.schema.json).
2. If the tool needs custom install logic (not just a package install), add a Go handler under `internal/tools/<name>/`, register it in `internal/runtime/registry.go` `customToolHandlers`, and reference the handler name in the manifest.
3. Reference the tool from the consuming `service.json` or `resource.json` `hostTools[]` array using a `hostRequirement` entry.
4. Verify with `go test ./internal/runtime/...` that the manifest-vs-handler invariant passes.

The wizard surfaces the new tool automatically once the manifest and any consumer references are in place.

## See also

- [`safeguards.md`](safeguards.md) — host-state-modifying counterparts to tools
- [`../architecture.md#resolution-order`](../architecture.md#resolution-order) — opt-in resolution rules
- [`internal/runtime/registry.go`](../../../internal/runtime/registry.go) — the canonical handler map
