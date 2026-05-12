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

## `web-console session`
[CODE: cli/domains/session/register.go]

| Subcommand | Description |
|---|---|
| `list` / `ls` | List active sessions |
| `get` / `show` | Show one session |
| `create` | Create a session (optionally `--body-file PATH`) |
| `delete` / `rm` | Terminate a session |
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
