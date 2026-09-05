# Onboarding handoff contract

`vrooli setup` completes bootstrap and then decides how the operator reaches
`vrooli-onboarding`. The decision is based on the invoking session, not on
whether the machine has a desktop installed.

`--onboarding` accepts five values:

- `auto`: open a browser only for a reachable local graphical, WSL graphical,
  or remote-desktop session; otherwise print the URL and resume command.
- `browser`: request a browser explicitly; a failed open falls back to the URL.
- `cli`: run the terminal wizard only when standard input is a terminal.
- `url`: start onboarding and print the URL without opening a browser.
- `none`: do not start or hand off to onboarding.

`VROOLI_SKIP_ONBOARDING=1` is the environment form of `--onboarding=none`.

Presentation detection reports `local-graphical`, `wsl-graphical`,
`forwarded-graphical`, `remote-desktop`, `remote-shell`, `headless`, or
`unknown`. Detection is bounded and never accesses a keyring. A degraded
probe is treated as unreachable, so the safe result is a printed URL.

Configuration completion has one authority: the project-state
`.configuration-complete` marker. The bootstrap marker only proves that the
non-interactive setup phase ran; it does not suppress onboarding.

The result-file payload remains version `v1`. Its optional `onboarding` object
records the decision, presentation kind, URL, resume command, and whether a
browser was opened.
