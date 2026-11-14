# Operations

`codex` is organized as an `external-cli` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Codex-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- `lib/` contains retained shell behavior only until the resource is fully migrated.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized binary discovery, install translation, version parsing, config-path handling, or auth validation, grow `cli/internal/discovery`, `cli/internal/install`, `cli/internal/version`, `cli/internal/env`, or `cli/internal/auth` first.

## Operator Checklist

- Keep install guidance and minimum supported version declared in `resource.json`.
- Route mutable config and runtime files through canonical resource storage instead of repo-local ad hoc paths.
- Separate binary discovery from login/config validation.
- Prefer shared lifecycle and invoke behavior before adding resource-local commands.

## Remote continuity ownership

The Codex resource owns CLI availability. Its healthy status does not establish
that a phone can reach the ChatGPT desktop app. The desktop app's app-server and
the shared CLI daemon are separate execution owners. They can share credentials,
installation identity and remote enrollment while exposing different local
transports and different loaded threads.

Identify the actual remote owner before changing a daemon. The CLI daemon's
`remoteControl/status/read` standing applies to that daemon. A disabled standing
there does not mean that the desktop app's separate remote connection is disabled.
Use `codex doctor --json`, `codex app-server daemon version`, the desktop app's
logs, and exact process identities to distinguish them. On the qualified local
versions, the protected Unix control socket uses WebSocket framing; a newline
JSON connection or a bare HTTP request cannot establish its health. Native
structured logs attribute remote connection events by `process_uuid`, including
the PID. Retain only the relevant redacted events, never token or pairing values.

The supported `enable-remote-control` and `disable-remote-control` daemon commands
can replace the managed daemon. Do not infer an uninterrupted update from help
that says the setting applies to a currently running daemon. Before maintenance,
retain the exact owner PID/start identity, complete loaded-thread statuses,
descendant inventory and persisted setting. Preserve active or unknown work.
Afterward, distinguish a restored idle thread or retained history from an executor
that ran without interruption. Metadata-only `thread/read` must not be replaced
with `thread/resume` merely to make the inventory look continuous.

Keep one active remote-control owner for an installation. A second daemon can
receive HTTP 409 with `Remote app server already online` while the desktop app
is connected. That is an ownership conflict, not evidence that credentials or
pairings should be deleted. Resolve the competing owner through its supported
operation after checking work preservation; do not alter the desktop app or
revoke devices as incidental cleanup. New human-only device approvals remain
human decisions.

`codex app-server daemon bootstrap --remote-control` is the upstream durable-host
setup surface. Qualify its actual native ownership and recovery behavior before
using it alongside an existing desktop owner. A PID-managed daemon inside an
agent's transient scope is not proof of boot recovery. The external CLI resource
does not provide a second private watcher or claim that scope as an independently
managed service. Keep the actual app, its agent scopes and unknown identities
protected until the supported handoff has been established.

For a Linux host whose phone connection is owned by the desktop application,
use the packaged desktop launcher through the user's XDG application autostart
configuration. This starts the existing app after an authenticated graphical
login; it does not require a competing CLI daemon or a new watcher. Derive the
entry from the installed package, retain the exact content, validate it with
`desktop-file-validate`, and preserve a user entry that already exists. A fixed
absolute packaged launcher path avoids selecting another executable from PATH.

The [XDG Autostart specification](https://specifications.freedesktop.org/autostart/latest/)
defines that login boundary. Login/unlock policy remains a separate choice.
An installed entry proves configuration only: next-login, physical boot, app
crash restart and actual phone access each require their own operated witness.
Do not launch or restart the desktop app merely to turn those evidence fields
green while it owns live threads.
