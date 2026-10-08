# vrooli-emulator runbook

## Supported distros

Phase 1 verified on:

| Distro            | Verified  |
|-------------------|-----------|
| Ubuntu 22.04 LTS  | 2026-04-25 |
| Ubuntu 24.04 LTS  | 2026-04-25 |

Required host packages (via `apt-get install -y`):

```
xvfb x11vnc websockify xdotool xclip procps ffmpeg x11-apps
```

`x11-apps` provides `xclock`, used as the launch fixture in the integration suite. `procps` provides `pgrep`, used by the post-teardown stray-process assertion.

## Running integration tests

The integration suite exercises the real `LinuxBackend` (Xvfb + x11vnc + websockify) against an in-process `httptest.Server`. It is gated by the `integration` Go build tag so it does not run in the default unit phase.

```
cd scenarios/vrooli-emulator/api
go test -tags=integration -timeout 300s ./livedesktop/...
```

When any of the required host binaries are missing the suite exits 0 with a skip message.

`vrooli scenario test vrooli-emulator` does not currently pass `-tags=integration` to `go test`; until the runner gains build-tag support, run the command above directly.
