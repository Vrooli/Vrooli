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

Manage Codex Bash rules and native execution intent through `permissions`. User configuration honors `CODEX_HOME`, defaulting to `~/.codex`. V1 documents manage only `[vrooli.permissions]` and preserve native settings. V2 execution documents additionally manage the reserved `[permissions.vrooli]` profile; activation explicitly selects it and updates approval settings. Admin scope resolves system `requirements.toml` (`/etc/codex/requirements.toml` on Unix, `%ProgramData%/OpenAI/Codex/requirements.toml` on Windows). Vrooli Bash metadata there is not native administrative enforcement; v2 execution projection rejects admin scope. Existing user-local `requirements.toml` is never silently migrated.

```bash
# Block git stash at user scope (motivating example)
resource-codex permissions deny 'git stash *'

# View managed patterns
resource-codex permissions list
resource-codex permissions show --raw

# Detect drift since the last Vrooli write
resource-codex permissions drift-check

# Check version and surface the enforcement caveat
resource-codex permissions doctor
```

Mutating verbs (`deny`, `allow`, `ask`, `remove`, `reset`) refuse agent callers (detected via `cliutil.DetectCallerKind`) unless `--i-was-explicitly-authorized` is passed. Read verbs are always allowed.

**Enforcement caveat.** Codex's effective native permission profile (or legacy sandbox) and approval settings remain authoritative; the `[vrooli.permissions]` section is a uniform policy projection rather than a native pattern matcher. Vrooli also projects `~/.codex/hooks.json` with a `PreToolUse` command hook when deny rules exist. The CLI reports this as `hook_unverified` until a live canary proves the installed Codex version fires and honors the hook; do not treat hook-file presence as sandbox enforcement.

For declarative automation, use `permissions plan --scope user|admin --document desired.json --json` and `permissions reconcile --scope user|admin --document desired.json --json`. The strict v1 document contains `schema_version`, matching `scope`, and ID-addressed `allow`/`ask`/`deny` rules with `matcher: {"kind":"bash","pattern":"..."}`. Plan never writes; reconcile is authorization-gated, preserves unmanaged TOML, and reports desired/live fingerprints, native paths, changes, and the `hook_unverified` enforcement posture.

### Native execution documents

`permissions capabilities` reports configuration support, independently of live
runtime evidence. Codex user scope requires an installed CLI >= 0.138.0 for offline profiles; filtered networking requires >= 0.156.1. Use the same optional `--codex-executable /absolute/path/to/codex` in plan and reconcile to qualify a specific installation. Version and command evidence appear in the preview. The
other resource adapters explicitly report unsupported execution projection;
Claude Code, OpenCode and Grok reject execution documents before any native write.
Antigravity exposes capabilities and retains its existing native CRUD surface.

A complete v2 example (network allowlists govern destinations, not API effects):

```json
{
  "schema_version": "v2",
  "scope": "user",
  "rules": [],
  "execution": {
    "filesystem": {"workspace": "write"},
    "network": {
      "enabled": true,
      "domains": {"localhost": "allow", "127.0.0.1": "allow", "::1": "allow"}
    },
    "approval": {"policy": "on-request", "reviewer": "auto_review"}
  }
}
```

`filesystem.workspace` is `read` or `write`. Optional `writable_roots` contains
clean absolute paths for the current OS, excluding the filesystem root. Added
roots retain read-only `.git`, `.codex`, and `.agents` descendants. Network
`enabled` is mandatory. Domains are exact lowercase hostnames or IP literals;
actions are `allow` or `deny`. Wildcards, URLs, ports, and domains on a disabled
network are rejected. Approval policy is `on-request` or `never`; reviewer is
`user` or `auto_review`, with automatic review requiring interactive approvals.
These settings do not modify app/plugin-specific approval controls.

```bash
# Preview staging: native profile and intent only, no active-default change.
resource-codex permissions plan --document desired.json --json
# Substitute the exact preview_digest returned above.
resource-codex permissions reconcile --document desired.json --expected-digest DIGEST --json

# Activation is a separate preview and reconciliation with matching flags.
resource-codex permissions plan --document desired.json --activate --json
resource-codex permissions reconcile --document desired.json --activate --expected-digest DIGEST --json
resource-codex permissions doctor
resource-codex permissions drift-check
```

Agent callers must also have explicit human authorization and pass
`--i-was-explicitly-authorized`. A digest is not authorization. Preview digests
bind the document, activation selection, and native config/hook snapshots,
including modes. Changed previews are rejected under the shared writer lock.
Activation enables the native network proxy so domain rules are enforceable.
Staged profiles keep network access disabled until activation, preventing manual selection from bypassing domain filtering.
Conflicting legacy sandbox settings or an unowned `permissions.vrooli` table
are rejected, not removed. Previously activated intent requires `--activate`
for subsequent execution reconciliation. Rule-only operations and reset retain
native execution intent. TOML values round-trip; formatting/comments may change.

Writers preserve POSIX modes and Windows access ACLs, create private new files, reject file symlinks,
and retain private pre-write config backups. Successful reconciliation returns
`recovery_backup` when a backup was needed. Hook-write failure attempts config
rollback and reports any partial hook migration. State publishes only after
native readback succeeds. On failure, inspect the named backup and current config,
then create a fresh preview; do not blindly restore over concurrent edits.
Backups may contain private configuration and remain until the operator removes
them. Reset clears Bash rules only; reverting execution settings requires an
explicit reviewed configuration migration or restoring the reviewed backup.

`doctor` distinguishes recorded activation intent, native projection equality,
hook registration equality and unverified effective runtime. Higher-precedence
project/profile files, launch flags and managed requirements can override this
file. Confirm a fresh desktop session's effective settings independently;
CLI version or config equality alone does not prove desktop adoption.

The installed native boundary can be qualified without a model or live user
config writes, using an explicit executable:

```bash
cd resources
VROOLI_CODEX_NATIVE_BINARY=/usr/bin/codex go test -v ./codex/cli/internal/permissions -run '^TestNativeExecutionSandboxCanary$' -count=1
```

The canary uses disposable config and files, checks permitted workspace writes,
blocked outside and read-only writes, allowed loopback HTTP, a denied hostname
against the same local server, direct socket bypass rejection and network-off
behavior, then cleans up its scratch directories. Windows ACL routines have
Windows-specific tests and cross-compilation coverage; execution on Windows must
be qualified on that host before claiming runtime verification. Python 3
and the native runtime's sandbox prerequisites are required. Runtime evidence is
specific to the tested executable, OS and version; it does not promote hooks or
the desktop client to verified.

Upstream docs: <https://learn.chatgpt.com/docs/permissions> and
<https://learn.chatgpt.com/docs/enterprise/managed-configuration>.


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


### Native compatibility evidence (2026-09-30 UTC)

The isolated Linux canary passes on `/usr/bin/codex` 0.156.1 for workspace,
read-only, loopback allow/deny, direct proxy bypass rejection and network-off
behavior. The user-local 0.141.0 runtime passes the offline canary but fails the
network-enabled loopback assertion with connection refusal. The adapter therefore
rejects filtered-network intent below the tested 0.156.1 floor, including staging;
it never falls back to unrestricted networking or edits the installed runner.
This is a conservative compatibility bound, not a claim about the earliest
upstream fixed release. Desktop behavior and Windows ACL execution remain
independently unverified.

Investigation hypotheses: (1) an unreachable native proxy listener; (2) the
resource profile losing its domain or feature settings; (3) host service or
namespace prerequisites. Running the same disposable profile and fixture on the
same host across two explicit executables rejects a general host/service outage
and supports a runtime-specific network path limitation. The offline .141 canary
also proves its filesystem projector works. The precise upstream proxy cause
remains unconfirmed. The application regression asserts that .141 network intent
is rejected before writing; it failed before the narrowed compatibility gate.
