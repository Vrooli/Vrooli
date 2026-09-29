# Codex Resource

OpenAI Codex CLI for local code generation and agentic engineering workflows.

## Intent

- Resource ID: `codex`
- Category: `developer-tooling`
- Driver: `external-cli`
- Portability tier: `partial`

## Use Cases

- Use Codex as an interactive or scripted coding agent in local workflows.
- Standardize Codex CLI availability for scenarios and operator tooling.
- Provide a consistent external CLI dependency for code generation and task execution.

## Architecture

This resource uses the updated `external-cli` structure.

- `resource.json` is the declarative authority for install, binary probing, version checks, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Codex-specific Go logic when the manifest and shared control plane are not enough.
- Historical shell behavior has been retired; lifecycle and configuration
  behavior lives in the shared control plane and typed Go packages.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Codex-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/discovery`: host binary detection and probing helpers
- `cli/internal/install`: install/bootstrap helpers
- `cli/internal/version`: version parsing and compatibility helpers
- `cli/internal/env`: environment and config-path helpers
- `cli/internal/auth`: auth/config validation helpers

## Usage

```bash
# Install using the declarative contract
vrooli resource install codex

# Check that the binary is available and healthy
resource-codex status
```

## Coding-role policy

Codex owns its concrete coding-role inventory in `model-policy.json`. Use
`resource-codex policy validate`, `policy roles --json`, and `policy resolve
--role code.default --json` to inspect it. The response records the concrete
model, fallbacks, policy provenance, and the intentionally `intent_only`
permission posture; Agent Manager consumes that response at run creation but
does not duplicate the inventory or write Codex configuration.

## Permissions

Linux namespace setup is owned by the control-plane `bubblewrap_userns`
safeguard, declared in `resource.json`. Use the [host repair and no-model
containment check](../../docs/configuration/host/safeguards.md#native-bubblewrap-namespaces).
The dedicated executable profile follows the scoped approach in
[OpenAI's sandbox guidance](https://learn.chatgpt.com/docs/sandboxing).
Do not switch to unrestricted execution to hide a sandbox startup failure.

Manage Codex bash-command patterns via the `permissions` subgroup. The adapter owns a Vrooli-namespaced `[vrooli.permissions]` section in `~/.codex/config.toml` (user scope) or `~/.codex/requirements.toml` (admin scope). All other Codex-native settings (`[profiles.*]`, `sandbox_mode`, `approval_policy`, …) round-trip untouched.

```bash
# Block git stash at user scope (motivating example)
resource-codex permissions deny 'git stash *'

# Same, admin-enforced
resource-codex permissions deny --scope admin 'git stash *'

# View managed patterns
resource-codex permissions list
resource-codex permissions show --raw

# Detect drift since the last Vrooli write
resource-codex permissions drift-check

# Check version and surface the enforcement caveat
resource-codex permissions doctor
```

Mutating verbs (`deny`, `allow`, `ask`, `remove`, `reset`) refuse agent callers (detected via `cliutil.DetectCallerKind`) unless `--i-was-explicitly-authorized` is passed. Read verbs are always allowed.

**Enforcement caveat.** Codex's native `sandbox_mode` and `approval_policy` remain the authoritative controls; the `[vrooli.permissions]` section is a uniform policy projection rather than a native pattern matcher. Vrooli also projects `~/.codex/hooks.json` with a `PreToolUse` command hook when deny rules exist. The CLI reports this as `hook_unverified` until a live canary proves the installed Codex version fires and honors the hook; do not treat hook-file presence as sandbox enforcement.

For declarative automation, use `permissions plan --scope user|admin --document desired.json --json` and `permissions reconcile --scope user|admin --document desired.json --json`. The strict v1 document contains `schema_version`, matching `scope`, and ID-addressed `allow`/`ask`/`deny` rules with `matcher: {"kind":"bash","pattern":"..."}`. Plan never writes; reconcile is authorization-gated, preserves unmanaged TOML, and reports desired/live fingerprints, native paths, changes, and the `hook_unverified` enforcement posture.

Upstream docs: <https://developers.openai.com/codex/permissions>.

## Model catalog operations

The operator's amendment permits Sol for infrequent, explicitly admitted
supervision through `judgment.supervision`. Ordinary roles still deny Sol; Astra
remains globally excluded. The owner-validated Codex catalog exposes
`gpt-6-luna` and `gpt-6-sol`, so `code.delivery` selects Luna medium and
`judgment.supervision` selects Sol medium from the subscription catalog. The
host can contain multiple Codex installations, so record the executable path
and `--version` with every catalog observation. On the current host,
`/usr/bin/codex` 0.156.1 lists GPT-6 Luna/Sol and the 5.6 models. The Vrooli
PATH may resolve an attribution shim and a different user-local installation.
Agent Manager's managed codec intentionally bypasses that shim and selects the
installed system runner (`/usr/bin/codex` or `/bin/codex` on Linux); a direct
shell `codex --version` is not managed-launch evidence. `~/.codex/models_cache.json`
is used only as a compatibility fallback. Older caches can omit valid model slugs and are
marked non-authoritative. The model-policy drift safeguard refuses to make
availability claims from that fallback, so an unavailable live probe is
reported as unmeasured rather than as a false missing-model finding. A measured
runner listing is also not assumed exhaustive: it proves a model is offered when
listed, but omission produces a non-blocking `unconfirmed_*` finding unless the
runner explicitly marks the catalog `exhaustive`. Those warnings must never be
reported as “GPT-6 is unavailable” or used to replace an owner-validated GPT-6
role with GPT-5.6. Catalog validation is not execution qualification. Keep the delivery team
disabled until its pilot succeeds and record the actual selected model.
Operational Codex roles now resolve the owner-validated GPT-6 Luna family;
`judgment.supervision` is the only ordinary role permitted to select GPT-6 Sol.
Reserved legacy aliases remain only for historical compatibility and are not
launch permissions. Keep legacy `model`/`fallbacks` and structured `models`
candidates consistent. Validate the resource policy and reload Agent
Manager's role policy after edits; inspect profile resolution to verify adoption
without buying inference. Existing run snapshots and manually selected native
sessions retain their original models. Stop an affected managed run through its
owner before any replacement. Hard deny rules are declared in the catalog's root
`excluded_models` list.
`restricted_models` maps a model to the only resource roles that may select it.
The resource resolves both rules into the existing `excluded_models` response,
including aliases; global denial always wins. New ordinary roles remain denied
without copying per-role lists. This controls model selection, not authority to
assume a role: the execution owner must bind workers and supervisors to their
admitted roles and enforce review cadence. Existing heartbeats do not become
Sol runs just because the new role is available.
Provider prices and subscription quota are separate measurements: do not invent
token prices or treat an estimated zero-dollar charge as unused weekly allowance.

`resource-codex models list --json` queries `codex debug models` without invoking a model and falls back to the Codex model cache only when the runner probe is unavailable. A live result is authoritative evidence that listed models are offered, not an exhaustive inventory; only `exhaustive: true` permits an absent-model conclusion. Record the measured binary path/version and verify managed launch behavior through the codec's shim-bypass test and bounded runner probe; a direct PATH lookup is not sufficient evidence. Never downgrade owner-validated GPT-6 roles from a stale cache or partial surface. `resource-codex models resolve --model <id> --json` returns the resource-owned canonical pricing identity. Run `resource-codex policy validate --against-live --json` after a retarget. `observed_at` has a 14-day budget; aliases should remain runner-facing while pinned fallbacks must be refreshed from the same live evidence. Policy edits are reviewed explicitly and are never made by the drift safeguard.

If this command reports an unexpectedly old partial catalog, rebuild the installed resource CLI from the current resource source before diagnosing model availability; a stale installed CLI can preserve obsolete discovery behavior.

## Notes

- `codex` is an external CLI resource, not a local daemon owned by this resource.
- Keep binary/version/install behavior declarative in `resource.json` whenever possible.
- Keep `cli/main.go` thin; do not treat it as the implementation surface for Codex-specific behavior.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/codex/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.

## Historical integration summary

The shell-era implementation summary has been retired from current guidance.
Its references to shell libraries, model prices, and automatic fallback behavior
are historical, not a description of the current resource implementation.
The original remains beneath runtime home at
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/resources/codex/docs/IMPLEMENTATION-SUMMARY.md`.
Use this README and the resource's current command help for supported behavior.
