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

## Who decides a model

Change a model in one place, `resources/<harness>/model-policy.json`, then apply it. Everything else reads from there.

| Source of truth | What it decides | What reads it |
|---|---|---|
| `roles` | The model and effort of each role (`code.default`, `code.cheap`, ...) | Agent Manager run creation, `resource-<harness> policy resolve`, Web Console model choices |
| `excluded_models`, `restricted_models` | Models no role may use, and models only named roles may use | Agent Manager admission, usage mode target validation, the policy validator |
| `interactive_default` | The model and effort of a session started with no choice (Codex: `gpt-6.1-sol`, `high`) | `resource-codex settings apply` / `ensure` writes it into `~/.codex/config.toml` (`model`, `model_reasoning_effort`) through `native-settings.json` `value_from`; Web Console "full power" |
| `usage_tiers` | What a usage mode lowers each model to, the lowered session model, exempt roles | The launcher shim, Agent Manager run overlay, Web Console, read live each time a mode is applied |
| A usage mode (`~/.vrooli/state/vrooli/usage-mode/mode.json`) | Only intent: kind, scope, expiry, extra exempt roles, an optional override model, effort, context limit | Same readers as `usage_tiers`; see [usage modes](usage-modes.md) |

`~/.codex/config.toml` is output, not input: never edit its `model` lines by hand. A value you set yourself there is reported as a conflict by `resource-codex settings plan` and left alone (`settings apply --force` replaces it, `settings decline interactive-model` stops managing it).

Changing the interactive default: edit `interactive_default` in `resources/codex/model-policy.json`, then run `resource-codex settings plan` and `resource-codex settings apply`. The policy validator rejects an excluded or unknown model. `restricted_models` limits which roles may resolve to a model; the interactive default is not a role, so a restricted model such as Sol may be the default.

## Choosing a model for a sub-agent

One rule: **do not name a model.** When you spawn a Codex sub-agent natively (`spawn_agent`), omit `model` and `reasoning_effort`. The child inherits the session's model, and the launcher already chose that from role policy (`resources/<harness>/model-policy.json`, roles such as `code.default` and `code.cheap`) and any active [usage mode](usage-modes.md). Nothing else needs to be known.

- Never pass `gpt-6-astra` or any other model by hand. It bypasses role policy (astra is an excluded model) and the usage mode. Two research bursts on 2026-11-08 and 2026-11-09 used about 20% of a weekly Codex quota in about an hour on astra at `max` effort.
- To run work on a specific role, start an Agent Manager run with `--role <role>`. Agent Manager applies policy, the usage mode and exclusions.
- To see the current model for a role: `resource-codex policy resolve --role <role>` (policy default; an active usage mode then lowers new work, see `agent-manager usage mode status`).
- If a task truly needs a different tier, tell the operator rather than overriding.

Enforcement on native spawn: Codex raises `PreToolUse` for `spawn_agent` (verified on Codex 0.167: `tool_name` is `collaborationspawn_agent`, and `tool_input.model` carries the named model). `vrooli-policy-runner model-hook --runner codex` is the guard for it. It denies (exit 2) only when it read the call and the policy cleanly and the named model is in `excluded_models` (by name or alias), and appends the attempt to `<agent-policy home>/excluded-model-spawns.jsonl`. Every other outcome allows the call and prints one warning. The guard is declared once in `resources/codex/managed-hooks.json` and projected into `~/.codex/hooks.json` by `resource-codex ensure` (run by `vrooli setup`) or `resource-codex hooks apply`. `resource-codex hooks plan` is the read-only form: it shows what would be written, touches only the identified Vrooli entries (operator hooks are never changed), and `permissions drift-check` fails if a declared hook is missing or edited. The command is the absolute installed path (`~/.vrooli/bin/vrooli-policy-runner`), so it does not depend on the PATH Codex started with; a missing binary or a hook that exceeds its 10-second timeout lets the call run. `make install` places the runner; until the installed runner is current the hook is inert and fails open.

**One operator step: trust the hook.** Codex does not run a hook until its owner trusts it. The grant is recorded in `config.toml` under `hooks.state` (keyed by hook position, hashed by content, so editing a hook revokes it), and Vrooli never writes it: it is a human security decision. Open Codex, run `/hooks`, review the `vrooli-model-hook` entry and trust it. `resource-codex hooks plan`, `hooks apply`, `ensure` and `permissions doctor` ask Codex itself (`codex app-server`, `hooks/list`; no model call) and print `trust: NOT TRUSTED ...` until it is done, or `trust unknown` if Codex cannot be asked. In a test, `codex exec` ran no hook at all until `--dangerously-bypass-hook-trust` was passed. The same applies to the deny-rule hook `resource-codex permissions` projects. Until the hook is installed, current and trusted, the instruction above is the only control.

How Codex reads a hook (verified on Codex 0.167 with a throwaway `CODEX_HOME`): only exit 2 blocks, with stderr as the reason; any other exit status is a non-blocking failure. A hook's stdout is parsed as Codex's own hook-output JSON, so Vrooli's decision document there marks every call `PreToolUse Failed`. For `--runner codex`, `vrooli-policy-runner hook` therefore blocks with exit 2 and a stderr reason, writes nothing to stdout, and treats `ask` as allow with a stderr note (Codex hooks cannot prompt). Other runners keep the decision document on stdout and the stable exit classes (10 ask, 20 deny); their native blocking convention is not verified here. A shell command reaches hooks as tool `Bash` (not `shell_command`), an edit as `apply_patch`, an MCP call as `mcp__<server>__<tool>`; the projected deny-rule matcher covers all of them. A crash inside `hook` is recovered and allowed, because a Go panic exits 2, which Codex reads as block.

## If it fails

A model-policy, usage-mode or settings fault never stops a coding agent from starting. Launchers fall back to the harness's own default and say so once on stderr; only an explicit request for something policy forbids is refused. Problems surface where an operator looks, not as failed launches: the `model_policy_drift` safeguard, `resource-<harness> policy validate`, and `resource-<harness> settings plan`.

| Fault | Interactive launch (`vrooli agent launch`, shim, Web Console) | Agent Manager run | Where it shows |
|---|---|---|---|
| Policy JSON missing, truncated, invalid, oversized or wrong schema | Launches on the harness default; if a usage mode is active, one stderr line and the mode keeps what it stored | `policy resolve` fails: the run is refused with the resource command's own message (no silent model) | drift safeguard `policy_invalid`; `policy validate` |
| `interactive_default`, a role or a usage tier names a model the runner no longer lists | Launch is never blocked (the harness reports an unknown model itself) | Run creation fails fast at the first real invocation with the runner's "model unavailable" classification; no automatic fallback for code roles | drift safeguard `missing_*` / `unconfirmed_*` findings (`interactive_default_model`, `usage_tier_model`, role models) |
| Excluded model referenced by a role | Not applicable | Role resolution skips excluded models; with none left the run is refused naming the exclusion | `policy validate` |
| Excluded model named explicitly to Agent Manager | Not applicable | Refused (the one deliberate refusal) | run error |
| Excluded model named in a native `spawn_agent` | Denied only by the trusted `model-hook` when it ran cleanly; any hook error allows | Not applicable | `excluded-model-spawns.jsonl` |
| Usage mode file corrupt, expired, unknown fields, wrong schema | Treated as no mode | Treated as no mode | `agent-manager usage mode status` |
| `settings apply` or `ensure` with an unusable policy | Not a launch path | Not a launch path | Settings that follow the policy are `held` (live value untouched, nothing retired), the others still apply, and the report notes the policy fault |
| `config.toml` unparsable, or the settings sidecar corrupt | Not a launch path | Not a launch path | `ensure` reports it and writes nothing; every write is atomic with a backup |
| Resource policy CLI hangs | Not a launch path | Resolution is cut off after 30 seconds and the error names the command | run error |
| A hook or checker errors, times out or is missing | Allowed. Codex 0.167 allows a call whose hook exits 1 or 20, or whose hook binary is missing (tested); only exit 2 blocks. A panic in `hook` or `model-hook` is recovered and allowed. A warning goes to stderr | Same | stderr warning |

## AI use classes

Provider eligibility is based on the verified reason for a request, not a class supplied by the request. The four classes are `own_account`, `bounded_feature`, `in_app`, and `harness_access`. Provider terms and explicit operator-approved rule records govern whether a route is eligible. An agent may propose or explain a rule; it cannot approve the rule, set trust, or set a price.

## Related contracts

- [AI provider contract](../resources/ai-provider-contract.md) owns provider behavior and policy integration.
- [Host tools](host/tools.md) owns discovery and managed updates for local coding-agent tools.
- [Paid features](../concepts/PAID_FEATURES.md) owns the application feature metering boundary.
- [Deployment packaging strategies](../architecture/packaging-strategies.md) owns desktop dependency packaging.
