# Operations

`antigravity` is organized as an `external-cli` resource (binary `agy`).

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, health, and
  `durable_data` metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Antigravity-specific Go logic that cannot be expressed
  through the manifest or shared control-plane packages:
  - `internal/upstream` — release-manifest version fetcher for `upstream-check`.
  - `internal/permissions` + `internal/permissionscli` — the native settings
    `permissions` adapter + governance command surface.

Do not turn `cli/main.go` into the implementation surface. Install and binary
download live in `lib/install.sh` / `lib/common.sh`.

## Install model

- Strategy: query the upstream per-platform JSON manifest
  (`<auto-updater>/manifests/<platform>.json` → `{version,url,sha512}`), then
  download the artifact it points at directly (the opencode/grok pattern) — not
  the official `curl | bash` installer. linux/macOS artifacts are `*.tar.gz`
  containing a binary named `antigravity` (renamed to `agy`); Windows artifacts
  are a bare `*.exe`. The sha512 from the manifest is verified before install.
- We deliberately do **not** run `agy install` afterwards: that step mutates the
  operator's shell profile (PATH append + alias purge). `~/.local/bin` is already
  on PATH.
- Target: `~/.local/bin/agy` (user-writable; updates never need root).
- Root-owned copy: refused with an actionable migrate message; never clobbered.
  `sudo` is reserved for privileged setup vacating such a copy.
- `uninstall` removes only the user-owned binary and **leaves `~/.gemini`**
  intact (durable config/permission/conversation state — removing it is the
  operator's deliberate choice, not an uninstall side effect).
- Reproducible/air-gapped installs: `ANTIGRAVITY_ARTIFACT_URL` (and optionally
  `ANTIGRAVITY_ARTIFACT_SHA512`) bypass the manifest lookup. No musl build is
  published; musl hosts are refused early.

## Version pin vs. self-update

`upstream_cli.version_pinned` is the **known-good baseline** for drift reporting,
not a forced install version: the manifest only serves "latest" and the artifact
URL embeds an opaque build id that cannot be reconstructed from a version string.
Additionally, `agy` self-updates in the background during normal runs, so the
installed version is a floor. `resource-antigravity upstream-check` reports
installed-vs-latest drift against the manifest.

## Permissions enforcement seam

Antigravity reads permission grants from the native `permissions` object in
`~/.gemini/antigravity-cli/settings.json`. There is **no user-writable hook
backstop** (its hooks are compiled-in), so this settings object is the single
enforcement seam — branch (a) "native rule file" of the resource plan. The
adapter owns only the managed `deny`/`ask`/`allow` arrays and preserves every
other settings key.

Schema (confirmed 2026-06-29 against `antigravity.google/docs/cli-permissions` and
the settings `agy 1.0.13` writes on first run): the `permissions` object holds
`allow`/`deny`/`ask` string arrays, evaluated **Deny > Ask > Allow**. Each entry
is an `action(target)` rule — `command(rm -rf)`, `read_file(*)`, `mcp(*)`, etc.;
`*` is a namespace wildcard. The adapter is vocabulary-agnostic (stores whatever
rule string is passed), so it maps 1:1 onto the native arrays. `permissions
doctor` documents this; a live canary (a denied command refused inside an `agy`
session) still proves end-to-end enforcement.

> **Confirmed paths.** Running `agy` creates the CLI's own state under
> `~/.gemini/antigravity-cli/` (its `settings.json` holds `trustedWorkspaces`,
> `permissions`, etc.; conversations are SQLite `.db` files under
> `conversations/`). This is **distinct** from the Antigravity 2.0 desktop app,
> which stores agent settings as protobuf (`~/.gemini/antigravity/user_settings.pb`)
> and shares only the top-level `~/.gemini/settings.json` (auth) and OAuth
> credential files. The permissions adapter correctly targets the CLI path.

## Operator Checklist

- Keep upstream binary/install/version expectations declared in `resource.json`.
- Keep durable host state declared in `durable_data` (host-filesystem only);
  mark credential entries `sensitive: true`.
- Separate auth/config validation from raw binary detection.
- Prefer shared lifecycle and invoke behavior before adding resource-local
  commands.

## agent-manager runner integration (NOT wired — deliberate follow-up)

This resource provides **install + update + backup + shared permissions + caller
detection** parity with the other coding agents (claude-code, codex, opencode,
grok). It does **not** register Antigravity as an `agent-manager` runner.

That integration — `RUNNER_TYPE_ANTIGRAVITY` proto enum, `domain.RunnerType`, a
`codecs/antigravity.go`, `main.go` registration, the env denylist, and the model
registry — is a separate follow-up plan, exactly as the Grok runner work
(`add-grok-runner-harden-agent-manager-codec-layer`) followed the Grok resource.
The headless surface for that future work is `agy -p "<prompt>" [-m <model>]`
(`--print`), with `--dangerously-skip-permissions` as the auto-approve posture.
There is deliberately **no `resource-antigravity run` passthrough**.
