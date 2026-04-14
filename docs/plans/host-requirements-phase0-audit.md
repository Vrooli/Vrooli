# Host Requirements Phase 0 Audit

**Status:** Approved Phase 0 output

**Last Updated:** 2026-04-12

## Purpose

This document is the concrete Phase 0 audit artifact required by
[host-requirements-declaration-and-runtime-redesign-plan.md](/home/matthalloran8/Vrooli/docs/plans/host-requirements-declaration-and-runtime-redesign-plan.md).

It freezes the current ground truth, ownership model, declaration contract, CLI
selection semantics, and runtime package boundaries needed before Phase 1
implementation begins.

## Current Ground Truth

- [`internal/runtime/runtime.go`](/home/matthalloran8/Vrooli/internal/runtime/runtime.go) still owns a fixed five-tool runtime model: `docker`, `go`, `node`, `python`, `helm`.
- [`internal/setup/setup.go`](/home/matthalloran8/Vrooli/internal/setup/setup.go) still calls that runtime layer directly; there is no manifest-driven resolver yet.
- Root [`.vrooli/service.json`](/home/matthalloran8/Vrooli/.vrooli/service.json) still shells out to `scripts/lib/setup.sh` via the `base-setup` lifecycle step.
- [`internal/lifecycle/lifecycle.go`](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle.go) still falls back to `scripts/lib/setup-conditions/<type>-check.sh` for unknown setup condition types.
- The shell-era setup layer still installs a broader host surface in the historical `scripts/lib/setup.sh` path, including `git`, `curl`, `jq`, `buf`, `sqlite`, `shellcheck`, `bats`, `lychee`, `ast-grep`, `js-yaml`, `ajv`, `tmux`, `yq`, `ffmpeg`, `Xvfb`, `xdotool`, `x11vnc`, `websockify`, and `openbox`.
- Shared shell helper debt is still live:
  - `55` scenario package manifests/scripts reference `scripts/lib/ui-guard.sh`
  - `64` scenario CLI installers reference `scripts/lib/utils/cli-install.sh`
  - `8` shell setup-condition checkers still exist under `scripts/lib/setup-conditions/`

## Approved Ownership Rules

1. Root manifest ownership is limited to universal bootstrap requirements and host-profile safeguards that apply to the machine as a whole.
2. Scenario manifests own scenario-specific host tools needed to build, run, test, or package that scenario.
3. Resource manifests own resource-specific host tools needed to provision, operate, or validate that resource.
4. Host safeguards are not just tools with a different name. They model policy or protection applied to the host, and default ownership is root or a future host profile, not a random scenario.
5. A requirement becomes root-core only when the repo cannot realistically function in the relevant setup mode without it. Broad usefulness is not enough.

## Approved Declaration Contract

Phase 1 should add the same top-level declarations to root `service.json`,
scenario `service.json`, and `resource.json`:

```json
{
  "hostTools": [
    {
      "name": "docker",
      "required": true,
      "reason": "Run containerized resources",
      "when": ["setup", "develop"],
      "environments": ["development", "production", "minimal"],
      "notes": "Root-owned core tool"
    }
  ],
  "hostSafeguards": [
    {
      "name": "remote_session_protection",
      "required": false,
      "reason": "Protect remote Linux GUI sessions from avoidable failure",
      "when": ["setup"],
      "platforms": ["linux"],
      "manual": false
    }
  ]
}
```

### Required fields

- `name`
- `required`
- `reason`

### Approved optional fields for Phase 1

- `when`
- `environments`
- `platforms`
- `notes`
- `manual`

### Deliberate exclusions from declaration payloads

- installer commands
- package-manager-specific package names
- sudo policy
- host probing logic
- ad hoc shell fallback references

Those belong in the native runtime registry implementation, not in manifests.

## Approved CLI Selection Semantics

- Keep the current `--resources enabled|none|<csv>` contract.
- Add `--scenarios none|all|<csv>`.
- Default `--scenarios` to `none` in `vrooli setup` so setup behavior does not silently widen during migration.
- Defer profile declarations as a first-class selector surface until after the base resolver exists.
- Continue using `--environment development|production|minimal` as the top-level setup mode selector.

## Approved Reporting Model

The native setup planner and executor should emit one structured result per
resolved requirement with:

- `name`
- `kind` (`tool` or `safeguard`)
- `owner`
- `provenance`
- `status`
- `reason`
- `manual_action`

Approved status vocabulary:

- `install`
- `apply`
- `already_present`
- `skip_not_declared`
- `skip_unsupported`
- `skip_not_applicable`
- `manual_action_required`
- `failed`

## Approved Package Boundaries

- `internal/hostreq`
  - manifest loading inputs
  - merge and dedupe
  - selector logic
  - provenance capture
  - host-facing classification inputs for runtime
- `internal/runtime`
  - planner/executor orchestration
  - shared result model
  - host and package-manager detection helpers
- `internal/runtime/tools`
  - one implementation file per tool
- `internal/runtime/safeguards`
  - one implementation file per safeguard

`internal/runtime` should stop owning the canonical requirement list. That
belongs to declarations plus the resolver.

## Audit Table

This table is the Phase 0 source of truth for implementation ordering.

| name | type | current path or assumption | current live consumers | proposed owner | proposed classification | native implementation needed | delete/replace/defer decision |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `docker` | tool | hardcoded in `internal/runtime`; shell-installed in `scripts/lib/setup.sh` | root setup; manifest-backed docker-service resources | root project | `core` | yes | keep and migrate first |
| `git` | tool | shell-era common dependency; native setup already configures git | root bootstrap; repo hygiene | root project | `core` | yes | keep |
| `curl` | tool | shell-era common dependency | shell resource/setup flows; network and download helpers | root project | `core` | yes | keep |
| `jq` | tool | shell-era common dependency | shell manifest parsing across setup/resource helpers | root project | `core` | yes | keep |
| `node` | tool | hardcoded in `internal/runtime`; shell-installed in `scripts/lib/setup.sh` | root JS/TS workspace; many scenario UIs and package scripts | root project | `core` for `development`; not core for `minimal` | yes | keep |
| `go` | tool | hardcoded in `internal/runtime`; shell-installed in `scripts/lib/setup.sh` | native CLI/control-plane build and many Go scenarios | root project | `core` for `development`; not core for `minimal` | yes | keep |
| `python` | tool | hardcoded in `internal/runtime`; shell-installed in `scripts/lib/setup.sh` | repo validation helpers and Python-based scenario/resource flows | root project | `core` for `development`; declared elsewhere if scenario-specific | yes | keep, but not global/minimal |
| `helm` | tool | hardcoded in `internal/runtime`; shell-installed in `scripts/lib/setup.sh` | packaging/deployment flows only | root or scenario manifests | `declared tool` | yes | keep, remove from implicit core |
| `tmux` | tool | optional shell-era common dependency | operator workflows; active implementation work appears scenario-specific rather than repo-critical | scenario manifests or future profile | `declared tool` | later | defer from early migration |
| `yq` | tool | optional shell-era common dependency; ad hoc binary installer exists | shell YAML utilities in `scripts/resources` and `scripts/lib/service` | scenario/resource manifests only if still needed | `declared tool` | later | defer; do not make core |
| `stripe` | tool | not part of native runtime; appears as scenario/business tooling | payment and webhook-oriented scenarios | scenario manifests | `declared tool` | yes | keep and declare for landing-page scenarios/templates |
| `vault` | tool | not in native runtime; root also has a `vault` resource | host Vault CLI usage is retired in favor of `resource-vault` | none | `delete` | no | permanently retire host `vault` ownership |
| `buf` | tool | shell-installed via `scripts/lib/tools/buf.sh` | protobuf/codegen owners | root or scenario manifests | `declared tool` | yes | keep, but not default core |
| `sqlite` | tool | shell-installed via `scripts/lib/runtimes/sqlite.sh` | sqlite-using resources/tests and external-cli resource flows | resource/scenario manifests | `declared tool` | yes | keep as explicit non-core tool |
| `shellcheck` | tool | shell-installed dev dependency | shell lint/test workflows | root dev/test profile or scenario manifests | `declared tool` | yes | keep as non-core dev tooling |
| `bats` | tool | shell-installed dev dependency | shell scenario/resource tests | root dev/test profile or scenario manifests | `declared tool` | yes | keep as non-core dev tooling |
| `lychee` | tool | shell-installed dev dependency | docs validation workflows | root dev/test profile | `declared tool` | yes | keep as non-core dev tooling |
| `ast-grep` | tool | shell-installed dev dependency | structural search/refactor workflows | root dev/tooling profile | `declared tool` | yes | keep as non-core dev tooling |
| `js-yaml` | tool | shell-installed Node-based CLI | YAML helper flows in shell tooling | scenario/resource manifests if still needed | `declared tool` | later | defer and reduce reliance |
| `ajv` | tool | shell-installed Node-based CLI | JSON-schema validation helpers | root dev/test profile or explicit scenario owners | `declared tool` | yes | keep as non-core tooling |
| `ffmpeg` | tool | shell-installed optional dependency | `scenario-to-desktop`, `web-console`, `browser-automation-studio`, `whisper` support flows | scenario/resource manifests | `declared tool` | yes | keep as specialized tool |
| `Xvfb` | tool | shell-installed Linux-only optional dependency | `scenario-to-desktop` display management; headed browser/server flows | scenario/resource manifests | `declared tool` | yes | keep as Linux-specific specialized tool |
| `xdotool` | tool | shell-installed Linux-only optional dependency | `scenario-to-desktop` proc/window detection | scenario/resource manifests | `declared tool` | yes | keep as Linux-specific specialized tool |
| `x11vnc` | tool | shell-installed Linux-only optional dependency | `scenario-to-desktop` live desktop/VNC bridge | scenario/resource manifests | `declared tool` | yes | keep as Linux-specific specialized tool |
| `websockify` | tool | shell-installed Linux-only optional dependency | `scenario-to-desktop` WebSocket VNC proxy | scenario/resource manifests | `declared tool` | yes | keep as Linux-specific specialized tool |
| `openbox` | tool | shell-installed Linux-only optional dependency | `scenario-to-desktop` virtual display window manager | scenario/resource manifests | `declared tool` | yes | keep as Linux-specific specialized tool |
| `remote_session_protection` | safeguard | historical shell safeguard idea only; no current native implementation | remote Linux/VPS deployment profile, not a specific scenario runtime | root host profile | `declared safeguard` | yes | implement in Phase 5 as a native sysctl/systemd safeguard |

## Adjacent Compatibility Surfaces To Migrate Later

These are live migration targets but not host requirements themselves.

| surface | current live evidence | phase interpretation | owner after redesign |
| --- | --- | --- | --- |
| `scripts/lib/setup.sh` | root `base-setup` still invokes it | legacy compatibility path to delete in Phase 6 | none |
| `scripts/lib/setup-conditions/*` | `internal/lifecycle` still shells out to them for unknown checks | native lifecycle debt to replace | lifecycle package |
| `scripts/lib/ui-guard.sh` | `55` scenario references | shared scenario helper debt, not host requirement | Phase 7 redesign target |
| `scripts/lib/utils/cli-install.sh` | `64` scenario references | shared installer helper debt, not host requirement | Phase 7 redesign target |

## Resolved Phase 0 Decisions

- Root core declarations live directly in `.vrooli/service.json`, not under a nested setup-only object.
- `python` remains root-owned for `development` setup, but not as a universal requirement across all environments.
- `tmux` is not core.
- `yq` is not core.
- `remote_session_protection` should be modeled first as a host-profile safeguard, not a scenario declaration.
- Host `vault` is retired; `resource-vault` is the only supported Vault CLI surface.
- Setup profiles are deferred until after the resolver exists; the declaration schema still allows `when` and `environments`.

## Phase 0 Exit Statement

Phase 0 is complete when this audit document and the main redesign plan are read
together and there is no unresolved ambiguity about:

- who owns a host requirement,
- where that requirement will be declared,
- whether it is core or explicit,
- how setup will select it,
- and whether the runtime must install, apply, skip, or defer it.
