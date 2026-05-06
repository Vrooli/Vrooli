# Buf Schema Registry (BSR) Login

This page documents the optional sign-in to the [Buf Schema Registry](https://buf.build) used by `path:packages/proto`.

## Status

**Optional.** Codegen runs without it. The token is only needed to refresh vendored proto modules (`make refresh-vendor`) — a manual operation that happens months apart, not during normal development.

The `vrooli-onboarding` v2 wizard (per [`OT-V2-FEATURE-COMPLETE`](../../../scenarios/vrooli-onboarding/PRD.md)) reads this folder and renders a connector card for buf-bsr automatically when integration-hub ships. Pre-V2, operators discover the optional login through `vrooli auth status` (see [host/tools.md](../host/tools.md)).

## What this is *not*

A blocker. After the proto-codegen pipeline switched to **local plugins + vendored modules**:

- `make generate` does **zero** outbound BSR requests.
- `make lint` does **zero** outbound BSR requests.
- `make breaking` does **zero** outbound BSR requests.
- A laptop on flight Wi-Fi, a CI runner with egress blocked, a fresh agent box with no BSR account — all generate code identically.

The only path that touches BSR is `make refresh-vendor`, which exports the latest `googleapis` and `protovalidate` modules into `path:packages/proto/vendor/`. Operators run it when intentionally upgrading those deps; nothing else does.

## Auth pattern

`external_sign_in_command` (see [`external-auth.md`](external-auth.md)).

```jsonc
"auth": {
  "kind":               "external_sign_in_command",
  "sign_in_command":    "buf registry login",
  "probe_path":         "$HOME/.netrc",
  "probe_match":        "machine buf.build"
}
```

The buf CLI v1.37 stores the token in `$HOME/.netrc` as a `machine buf.build` line. There is no `buf registry whoami` command in 1.37; presence of the `.netrc` line is the deterministic probe.

## Sign-in procedure

1. Visit https://buf.build/settings/user → **API tokens** → **Create token**.
2. Pick a token expiry policy (see [Token expiry policy](#token-expiry-policy) below).
3. Run `buf registry login`. Buf opens an access-code page in the browser; paste the token into the terminal when prompted.
4. (One-time) `chmod 600 ~/.netrc` to restrict the token file. The default permissions on most Linux distros leave it world-readable, which is fine for shared local repos but not for shared machines or hosts with multiple operators.
5. Verify with `vrooli auth status` (reports `buf: signed_in`) or by inspecting `grep '^machine buf.build' ~/.netrc`.

## Token expiry policy

Buf offers four expiry windows when creating a personal access token:

| Option | Recommended for | Notes |
|---|---|---|
| 1 month (default) | Nobody | Silently fails after 30 days; codegen pipelines lose the ability to refresh vendor modules with no warning. |
| 6 months | Rarely the right choice | Skip; pick 1 year if you want auto-rotation. |
| 1 year | **Security-conscious teams, shared/multi-operator hosts** | Annual rotation as routine housekeeping. Pair with a calendar reminder. |
| **Never expires** | **Solo operator on a personal box with `chmod 600 ~/.netrc`** | Recommended default. The token is read-only, used only for vendor refreshes. Auto-expiry buys minimal security and causes silent failures months later. |

Why "never" is the recommended default: with CD-1 (local plugins) and CD-2 (vendored modules), the BSR token is consulted **only** during `make refresh-vendor`. No write or publish scopes are requested. A leaked read-only token's worst-case impact is reading public BSR modules already published — nothing the attacker couldn't do anonymously. Auto-expiry would, by contrast, break a `buf export` months from now when the operator is mid-task.

If you do pick a fixed expiry, schedule the renewal:

```bash
# 11-month reminder for a 1-year token
vrooli schedule create --in 11mo "Renew BSR token before silent expiry"
```

## Renewal procedure

```bash
buf registry logout            # invalidates ~/.netrc entry
buf registry login              # paste new token
chmod 600 ~/.netrc              # idempotent
vrooli auth status              # verify signed_in
```

## Probe contract

The `vrooli auth status` command implements the [`external_sign_in_command`](external-auth.md#external_sign_in_command) probe contract for buf:

- **Default**: reads `$HOME/.netrc` for `^machine\s+buf\.build\b`. Match → `signed_in`. No match → `signed_out`.
- **`--check-expiry`**: additionally runs an authenticated read against BSR (e.g. `buf curl --schema buf.build/bufbuild/protovalidate -o /dev/null`). 401 / 403 → `expired`. Gated behind the flag because the default probe must not generate BSR traffic on every status check.

## When to actually use the token

After the local-plugin switch, the answer is **only when running `make refresh-vendor`**. The refresh script is the single code path that calls `buf export`, which fetches the upstream BSR modules into `path:packages/proto/vendor/`. Anonymous BSR access works for `buf export` until the rate limit kicks in; the token raises the ceiling.

Refresh cadence is "when needed" — typically months between bumps, when consuming code wants a new field from googleapis or a new validate constraint. Run it, commit the updated `vendor/` tree, and `make check` enforces the resulting `gen/` diff is the only churn.

## Related documents

- [`external-auth.md`](external-auth.md) — the `external_sign_in_command` pattern this page implements
- [`../host/tools.md`](../host/tools.md) — how `buf` and the proto plugins are declared as host tools
- [`../../development/proto.md`](../../development/proto.md) — proto codegen pipeline overview