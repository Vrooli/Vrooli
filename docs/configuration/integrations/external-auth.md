# External Authentication

Concept page for OAuth, device-flow, and coding-agent sign-in credentials. **Schema is intentionally deferred** until the first concrete integration is wired; this page exists to describe the framework, the gap, and the constraints on the eventual schema.

## Why this is different from paste-string secrets

Paste-string secrets (API keys) fit cleanly into the `secretDescriptor` shape documented in [`../secrets.md`](../secrets.md): the operator obtains a string from a provider's web UI, pastes it once, and the system stores it in Vault. The flow is non-interactive after the operator has the string.

External-auth credentials don't fit:

- **OAuth (web flow)** — operator clicks a button, gets redirected to the provider, authorizes, gets redirected back with a code. The system exchanges the code for a token. The "value" is not something the operator pastes; it's something the system receives.
- **OAuth (device flow)** — the system displays a code; the operator visits a URL, enters the code, authorizes; the system polls until authorization completes.
- **Coding-agent sign-in** — operator runs a sign-in command in a terminal (`claude /login`, `codex login`, etc.); the agent persists the token in its own config. The system needs to *detect* whether sign-in succeeded but doesn't manage the token directly.

These flows produce credentials, but the *obtain step* is interactive and provider-specific. A schema that flattens them all into "paste this string" loses the actual UX.

## Constraints on the eventual schema

When the first external-auth integration ships, the schema must:

1. **Distinguish the auth pattern** — at minimum `oauth_web`, `oauth_device`, `external_sign_in_command`.
2. **Carry the URL or command needed to initiate the flow** — for OAuth, the auth URL and (for web flow) the callback URL; for sign-in commands, the command string.
3. **Specify how the result is stored** — OAuth tokens go in Vault under a reserved prefix (`secret/vrooli/auth/<integration>` is the proposed convention); sign-in commands store tokens in their own config and the system probes to detect them.
4. **Specify how to probe for "is authentication still valid"** — for OAuth, a request to a known authenticated endpoint; for sign-in commands, a `<command> whoami` or equivalent.
5. **Reuse `secretDescriptor.classification` semantics** — `user` for operator-provided credentials remains correct.

The schema should *not* try to model:

- The provider's authentication flow steps in detail. Each provider varies; the schema declares which pattern, and Go handlers per provider implement the specifics. This is the same shape as host-tool / safeguard handlers.
- Refresh-token rotation policy. That belongs to the runtime code per provider, not to the manifest.
- UI button copy or icon. That's a frontend concern, not configuration.

## Proposed Vault layout

When the schema lands, the convention will be:

```
secret/vrooli/auth/<integration-name>
```

with keys like `access_token`, `refresh_token`, `expires_at`. This sits parallel to `secret/vrooli/<resource-name>` for paste-string secrets.

## Where this lives in the wizard (eventual)

The integrations step of the wizard will:

1. List external-auth-pattern integrations the operator's selected scenarios depend on.
2. For each, render a "Sign in with X" button (web OAuth) or "Run `command` then click here" prompt (sign-in command) or device-flow card (device OAuth).
3. Probe for completion and show green-light / actionable error.

Coding-agent sign-ins specifically are out-of-band: the package is installed during host-setup, but the sign-in is a separate step the operator runs in their terminal. The wizard's job is to *detect* successful sign-in (probe `<agent> whoami` or equivalent) rather than to handle the sign-in itself.

## Why this is deferred

We have zero external-auth integrations wired today. Designing the schema speculatively would paint us into a wrong shape on first contact with reality. Per the discipline in [`../README.md`](../README.md): build for one, generalize after three.

When the first integration is proposed (likely Cloudflare tunnels for `cloudflared` auth, GitHub OAuth for repository access, or coding-agent sign-in detection), this page becomes the schema-design conversation's starting point.

## See also

- [`../secrets.md`](../secrets.md) — paste-string credential pattern (handles all current integrations)
- [`../architecture.md`](../architecture.md) — open work items list, including this deferral
- [`README.md`](README.md) — discipline for adding new integration pages
