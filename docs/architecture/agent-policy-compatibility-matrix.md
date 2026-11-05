# Agent Policy Compatibility Matrix

| Consumer | Existing surface | New surface | Verification |
| --- | --- | --- | --- |
| Resource permission CLIs | `cli-core/agentpolicy` permission APIs | unchanged | resource unit tests and projection round trips |
| Claude Code | native permission document plus managed hook projection | `resource-claude-code permissions hook-guard` evaluates the shared runtime in process; native deny patterns remain a backstop | hook replay tests and a live stdin probe of the installed binary |
| Codex | native permission document | runner projection with native fallback | config tests and capability probe |
| OpenCode | native permission document/plugin projection | runner projection is explicit unverified until canary | truthful posture report |
| Antigravity | native permission document | runner projection with native fallback | config tests and capability probe |
| Grok | native permission document | runner projection with native fallback | config tests and capability probe |
| Provider scenarios | live scenario APIs | snapshot publication bridge | bundle contract tests |

The new runtime is intentionally additive at the resource boundary during
advisory rollout. No resource may claim a verified hook without a canary that
observes the hook firing and validates the returned decision and exit code.
Filesystem removal is the exception to the rollout. It is floor policy,
enforced for every runner under every profile (see
`agent-policy-runtime.md` §"Filesystem removal").


## Native execution projection

`permissions capabilities` is shared across all five resource CLIs. Its status
means configuration support, not verified live enforcement.

| Resource | v1 Bash document | v2 execution | Effective runtime proof |
| --- | --- | --- | --- |
| Codex | supported | user scope, offline >= 0.138.0; filtered network >= 0.156.1; explicit digest-bound activation | isolated no-model canary; desktop independently unverified |
| Claude Code | supported | rejected before writes | no native execution projector qualification |
| OpenCode | supported | rejected before writes | no native execution projector qualification |
| Grok | supported | rejected before writes | no native execution projector qualification |
| Antigravity | existing native CRUD, no document endpoint | unsupported capability | no native execution projector qualification |

Portable v2 intent must never be silently reduced to Bash rules. Adding support
for another adapter requires native equivalent boundaries, version qualification,
conflict handling, safe publication and runtime canaries. Shared intent does not
promise equivalent enforcement on an unqualified provider. The portable schema
currently excludes full-access grants, wildcard network destinations, arbitrary
native keys, administrative execution projection and plugin-specific approvals.
