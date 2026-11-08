# Coding agents

Coding-agent harnesses are tools a person uses to work with code. Vrooli helps discover and launch harnesses that are already installed, and provides a managed lifecycle where supported. A harness is separate from the model provider it uses: the harness supplies the coding workflow; the provider supplies inference.

## Three independent choices

| Choice | Meaning | Examples |
|---|---|---|
| Harness | The coding tool and its interaction model | OpenCode, Codex, Claude Code, Grok, Antigravity |
| Model source | Where inference comes from | The user's existing subscription, the user's provider key, local compute, or Vrooli credits |
| Deployment | Where the app and supporting services run | A full Vrooli installation, a downloaded desktop app, or a hosted deployment |

These choices have different support and trust rules. A deployment target does not determine which harness or model source a person may use.

## Existing and managed harnesses

The app can discover harnesses on the machine and show their installed version and lifecycle state. Discovery does not make an existing installation Vrooli-managed, and a discovered copy must not be silently renamed or replaced. Where the managed lifecycle supports it, users can install, update, repair, roll back, and remove the managed copy. Actions must explain when an installation method or filesystem owner prevents an operation.

The control plane owns host inspection and repair. Desktop packaging must not assume the full `vrooli` CLI is present; its embedded control surface is intentionally smaller. App surfaces should use the supported lifecycle API and report unsupported actions clearly.

## Model sources and keys

The user may use a harness's own subscription, configure a provider key, use a local model, or choose Vrooli credits where the harness and provider rules allow it. The selected model source is resolved for the launched session. Do not rewrite a user's global harness configuration to make a session work.

## Vrooli-managed native settings

A few harness settings are part of Vrooli's operating model rather than one session's needs. For example, Codex's multi-agent thread slots and sub-agent instructions make long, multi-level efforts possible. Each harness resource declares them in `resources/<harness>/native-settings.json` with a rationale. `resource-<harness> ensure`, which `vrooli setup` runs, applies them. Ownership is per key:
- Vrooli writes only keys that are absent or that still hold the value it last wrote.
- A value the user chose is reported as a conflict and left alone.
- `resource-<harness> settings decline <id>` opts a machine out.

Native files carry only keys the harness reads. Vrooli's own records live in sidecars beside the native file: `.vrooli-native-settings-state.json` (which native settings Vrooli wrote), `.vrooli-permissions-state.json` (the last permissions write, for `drift-check`) and, for Codex, `.vrooli-permissions-intent.json` (command-pattern rules and the execution document, which Codex has no setting for). Ownership markers (`_id`, `managedBy`) go only on individual hook entries; Codex refuses a `hooks.json` with any unknown top-level key. Before Vrooli rewrites a native file it keeps `<file>.vrooli-<writer>.orig` (the file before that writer's first rewrite) and `.bak` (the latest pre-image).

The shared rules live in `packages/agentharness` (`native_settings.go`, `native_file.go`). [Long-effort design](../agent-system/LONG_EFFORT_DESIGN.md) §4 records why each Codex value exists and how it was verified.

Treat any key provided to a coding agent as visible to that agent and to code it runs. A Vrooli-credits session uses a short-lived, scoped session key, not the user's account key. The launcher owns the session configuration and revokes the session key when that session ends.

Vrooli does not route Claude Code through Vrooli credits. Keep Claude Code's own-subscription and user-configured provider paths distinct from Vrooli-funded routing, and do not hide a supported sign-in method.

## AI use classes

Provider eligibility is based on the verified reason for a request, not a class supplied by the request. The four classes are `own_account`, `bounded_feature`, `in_app`, and `harness_access`. Provider terms and explicit operator-approved rule records govern whether a route is eligible. An agent may propose or explain a rule; it cannot approve the rule, set trust, or set a price.

## Related contracts

- [AI provider contract](../resources/ai-provider-contract.md) owns provider behavior and policy integration.
- [Host tools](host/tools.md) owns discovery and managed updates for local coding-agent tools.
- [Paid features](../concepts/PAID_FEATURES.md) owns the application feature metering boundary.
- [Deployment packaging strategies](../architecture/packaging-strategies.md) owns desktop dependency packaging.
