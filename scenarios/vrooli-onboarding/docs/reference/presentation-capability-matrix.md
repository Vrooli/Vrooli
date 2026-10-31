# Presentation capability matrix

| Kind | Platform and classification rule | Evidence tier | Upgrade command |
| --- | --- | --- | --- |
| `local-graphical` | Linux display/Wayland plus a non-remote login session; macOS Aqua; Windows active console session | unit-only | Run `vrooli setup --onboarding=auto` from a local desktop session and capture the result JSON plus browser journey |
| `wsl-graphical` | Linux WSL marker plus DISPLAY or WAYLAND_DISPLAY | unit-only | Run the same command inside WSLg and capture the browser journey |
| `forwarded-graphical` | SSH session with DISPLAY; reachable but auto policy prints a URL | unit-only | Run `ssh -X` setup and record the forwarded browser result |
| `remote-desktop` | Windows non-console, non-zero active process session | unit-only | Run setup from an RDP session and capture the result JSON |
| `remote-shell` | SSH/TTY session without a display | `hardware` | See [historical first-run evidence](#historical-first-run-evidence) |
| `headless` | CI/container override or no presentation signals | `container` | Run `vrooli setup --onboarding=auto` in a fresh container and capture output and markers |
| `unknown` | Unsupported operating system or unavailable probes | unit-only | Run setup on the target operating system and record the probe evidence |

The macOS and Windows rows are intentionally unit-only in this development
environment. The upgrade commands above are the required evidence, not claims
that those hardware runs occurred here.

## Historical first-run evidence

The 2026-08-24 capture is preserved beneath runtime home at
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/vrooli-onboarding/docs/evidence/first-run-handoff/20260824-221341/`.
It records an SSH remote-shell session on Linux. Local graphical and isolated
headless sessions were unavailable; macOS and Windows hardware were not measured.
The missing pre-change raw setup transcript remains missing; the immutable
Git Control Tower baseline is the recorded before-state authority.
The owner is `plan-manager plans get first-run-handoff-completion-truth-and-structural-cleanup`.
Evidence tiers in the table describe the historical qualification recorded here;
upgrade commands describe the additional observations required.
