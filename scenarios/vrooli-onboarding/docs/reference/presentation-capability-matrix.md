# Presentation capability matrix

| Kind | Platform and classification rule | Evidence tier | Upgrade command |
| --- | --- | --- | --- |
| `local-graphical` | Linux display/Wayland plus a non-remote login session; macOS Aqua; Windows active console session | unit-only | Run `vrooli setup --onboarding=auto` from a local desktop session and capture the result JSON plus browser journey |
| `wsl-graphical` | Linux WSL marker plus DISPLAY or WAYLAND_DISPLAY | unit-only | Run the same command inside WSLg and capture the browser journey |
| `forwarded-graphical` | SSH session with DISPLAY; reachable but auto policy prints a URL | unit-only | Run `ssh -X` setup and record the forwarded browser result |
| `remote-desktop` | Windows non-console, non-zero active process session | unit-only | Run setup from an RDP session and capture the result JSON |
| `remote-shell` | SSH/TTY session without a display | `hardware` | Captured under `docs/evidence/first-run-handoff/20260824-221341/` |
| `headless` | CI/container override or no presentation signals | `container` | Run `vrooli setup --onboarding=auto` in a fresh container and capture output and markers |
| `unknown` | Unsupported operating system or unavailable probes | unit-only | Run setup on the target operating system and record the probe evidence |

The macOS and Windows rows are intentionally unit-only in this development
environment. The upgrade commands above are the required evidence, not claims
that those hardware runs occurred here.
