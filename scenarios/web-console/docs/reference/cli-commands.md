# CLI Reference

The `web-console` CLI is a thin shell over the HTTP API. Command groups are registered in [CODE: cli/domains/domains.go]; each domain's verbs are defined in `cli/domains/<domain>/register.go`. All `--body-file` flags accept a path to a JSON file matching the corresponding [API endpoint](./api-endpoints.md) request body.

Run `web-console --help` for global flags (`--base-url`, `--timeout`, output format).

## Flat command groups

These are registered as flat groups (no subverb dispatch beyond a single noun).

### `web-console events`
[CODE: cli/domains/events/register.go]
Stream or list recent service events from `GET /api/v1/events`.

### `web-console metrics`
[CODE: cli/domains/metrics/register.go]
Fetch the snapshot returned by `GET /api/v1/metrics`.

### `web-console capabilities`
[CODE: cli/domains/capabilities/register.go]
Show capability inventory and liveness from `GET /api/v1/capabilities`.

## `web-console target`
[CODE: cli/domains/targets/register.go]

The target catalog is the supported way to discover remote session locations. It uses `TargetCatalogService`; it does not accept Bridge URLs or credentials from callers.

| Subcommand | Description |
|---|---|
| `list` | List local and remote locations, platform, readiness state, and dispatchability. |
| `get <target-id>` | Show the safe metadata for one location. |
| `doctor <target-id>` | Explain readiness facts and the first recovery action. |

Example:

```bash
web-console target list
web-console target doctor bridge-node:build-host
web-console session create \
  --target bridge-node:build-host \
  --working-dir /workspaces/project \
  --launch-command 'codex login --device-auth' \
  --execute-launch-command
```

Use `--json` with target commands for the lossless proto JSON projection. It contains no owner token, re-authentication proof, or Bridge endpoint.

## `web-console session`
[CODE: cli/domains/session/register.go]

| Subcommand | Description |
|---|---|
| `list` / `ls` | List active sessions |
| `get` / `show` | Show one session |
| `create` | Create a session (optionally `--body-file PATH`). Target/launch flags: `--target <target-id>` (use `target list`), `--working-dir <path>`, `--origin ui\|programmatic\|remote` (default `programmatic`), `--owner <tag>`, `--label <text>`, `--launch-command <cmd>`, `--execute-launch-command` (run the launch command immediately for local sessions, or once on first terminal attach for remote sessions), and `--idempotency-key <key>` (make retries replay-safe) |
| `delete` / `rm` | Permanently delete a session and its retained data |
| `archive` | Archive a session non-destructively |
| `unarchive` | Undo an archive by clearing its archive marker |
| `archive-list` | List collapsed archived-session lineages and restore states |
| `reopen` | Reopen an archived session with a required replay-safe idempotency key |
| `archive-retention` | Show archive entry counts, bytes, and configured retention limits |
| `archive-prune` | Preview retention actions; add `--apply` to execute them |
| `policy-get` | Show the expiration policy for a session |
| `policy-set` | Set the expiration policy (`--mode`, optional `--duration`) |
| `conversation` | Show the conversation feed for a session |
| `conversation-cursor` | Update the conversation cursor (`--body-file PATH`) |
| `summarize-event` | Trigger summarization of a conversation event |
| `list-recoverable` | List orphaned persistent sessions awaiting recovery |
| `recover` | Recover an orphaned persistent session into a fresh pane |
| `dismiss` | Permanently dismiss an orphaned session row (preserves on-disk state) |

## `web-console workspace`
[CODE: cli/domains/workspace/register.go]

| Subcommand | Description |
|---|---|
| `layout-get` / `layout` | Show the current workspace layout |
| `layout-save` | Save a workspace layout (`--body-file PATH`) |
| `pane-update` | Update a pane assignment (`--body-file PATH`) |
| `pane-delete` | Remove a pane assignment |
| `group-create` | Create a workspace group (`--body-file PATH`) |
| `group-update` | Update a workspace group (`--body-file PATH`) |
| `group-delete` | Delete a workspace group |
| `role-list` | List roles in a group, or every role. A role with no session id is *waiting*: it holds a command and no process |
| `role-create` | Create a role in a group (`--body-file PATH`). Omit `session_id` for a waiting role |
| `role-update` | Update a role (`--body-file PATH`). Setting `session_id` to `""` returns the role to waiting |
| `role-delete` | Delete a role. The session a running role points at is untouched |

## `web-console group-template`
[CODE: cli/domains/grouptemplates/register.go]

A template is a saved role list: creating a group from one creates the group and
its roles in a single action. Each role carries a `start_mode` of `eager` or
`waiting`, and only an `eager` role starts a process.

| Subcommand | Description |
|---|---|
| `list` | List saved group templates and their roles |
| `upsert` | Create or update a template (`--body-file PATH`) |
| `delete` | Delete a template. Every template is deletable, including a shipped example |

## `web-console handoff-rule`
[CODE: cli/domains/handoffrules/register.go]

A rule decides when the console *offers* a handoff. It never sends anything.

| Subcommand | Description |
|---|---|
| `list` | List capture rules |
| `upsert` | Create or update a rule (`--body-file PATH`). `source` is `file_path` (a glob) or `message_text` (a regular expression whose first capture group becomes the payload) |
| `delete` | Delete a rule. Every rule is deletable, including a shipped example |

## `web-console settings`
[CODE: cli/domains/settings/register.go]

| Subcommand | Description |
|---|---|
| `session-defaults-get` / `session-defaults` | Show default values applied to new sessions |
| `session-defaults-set` | Update default session settings (`--body-file PATH`) |

## `web-console shortcuts`
[CODE: cli/domains/shortcuts/register.go]

| Subcommand | Description |
|---|---|
| `effective` / `list` | Show effective shortcut bindings for this client |
| `profiles` | List saved shortcut profiles |
| `upsert` | Create or update a shortcut profile (`--body-file PATH`) |
| `delete` | Delete a shortcut profile |

## `web-console ai`
[CODE: cli/domains/ai/register.go]

| Subcommand | Description |
|---|---|
| `generate` | Generate a shell command (`--body-file PATH`) |
| `suggest` | Get AI suggestions (`--body-file PATH`) |
| `config-get` / `config` | Show AI provider config |
| `config-set` | Update AI provider config (`--body-file PATH`) |
| `health` | Check configured AI providers |

## `web-console voice`
[CODE: cli/domains/voice/register.go]

| Subcommand | Description |
|---|---|
| `config-get` / `config` | Show voice config |
| `config-set` | Update voice config (`--body-file PATH`) |
| `wakeword-get` | Show wake word config |
| `wakeword-set` | Upload/update a wake word template (`--body-file PATH`) |
| `wakeword-delete` | Delete the wake word template |
| `speaker-config-get` | Show speaker verification config |
| `speaker-config-set` | Update speaker verification config (`--body-file PATH`) |
| `speaker-status` | Show speaker verification status |
| `speaker-profiles` | List speaker verification profiles |
| `speaker-enroll` | Enroll a new speaker profile (`--body-file PATH`) |
| `speaker-clear` | Clear the bound speaker profile for this client |
| `speaker-remove` | Remove a speaker profile (`--body-file PATH`) |
| `speaker-delete` | Hard-delete a speaker profile (`--body-file PATH`) |

## `web-console tts`
[CODE: cli/domains/tts/register.go]

| Subcommand | Description |
|---|---|
| `config-get` / `config` | Show TTS config |
| `config-set` | Update TTS config (`--body-file PATH`) |
| `status` | Show TTS runtime status |
| `voices` | List available TTS voices |
| `summarize-config-get` | Show TTS summarize config |
| `summarize-config-set` | Update TTS summarize config (`--body-file PATH`) |
| `synthesize` | Synthesize speech (`--body-file PATH`, `--output PATH`) |
| `cache-get` | Fetch a cached synthesis by event id (`--output PATH`) |
| `event` | Post a TTS playback event (`--body-file PATH`) |
