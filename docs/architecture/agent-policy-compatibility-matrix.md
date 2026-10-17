# Agent Policy Compatibility Matrix

| Consumer | Existing surface | New surface | Verification |
| --- | --- | --- | --- |
| Resource permission CLIs | `cli-core/agentpolicy` permission APIs | unchanged | resource unit tests and projection round trips |
| Claude Code | native permission document plus managed hook projection | `vrooli-policy-runner hook --runner claude-code` where installed hook canary passes | native config preservation and canary |
| Codex | native permission document | runner projection with native fallback | config tests and capability probe |
| OpenCode | native permission document/plugin projection | runner projection is explicit unverified until canary | truthful posture report |
| Antigravity | native permission document | runner projection with native fallback | config tests and capability probe |
| Grok | native permission document | runner projection with native fallback | config tests and capability probe |
| Provider scenarios | live scenario APIs | snapshot publication bridge | bundle contract tests |

The new runtime is intentionally additive at the resource boundary during
advisory rollout. No resource may claim a verified hook without a canary that
observes the hook firing and validates the returned decision and exit code.
