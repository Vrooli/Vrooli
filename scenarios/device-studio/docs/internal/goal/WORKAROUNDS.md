# Infrastructure observations

- 2026-10-20T01:50:07.484838+00:00: `agent-manager run identity --json` failed with `agent authorization identity_refresh_unavailable`; supported `vrooli agent authorization <current session>` returned `[]`. No CreateRun admission attempted; no policy denial of filesystem work asserted. Do not retry with changed identity or anonymous headers. Direct native collaboration is the operator-authorized execution route; no background team/heartbeat claim. Revisit managed route only through supported identity recovery.

QA observation published as `knw-1792461091342972615` through report-bug; no root cause or recovery claim.

- 2026-10-20: BAS H.264/browser investigation (read-only; no launch, config change, install, restart, or lease mutation). The retained `p6b-ui/live-run1/codec-observer/observation.json` records BAS Chromium/Chrome 120 reporting no `canPlayType` support for the source's H.264 baseline/main/high variants, while VP9 was reported `probably`; this is a browser capability observation, not a full decode result. Host inventory reported `/usr/bin/google-chrome` -> `/opt/google/chrome/google-chrome`, Google Chrome 150.0.7871.128; `/usr/bin/firefox` is Firefox 157.0; cached Playwright Chromium 1169 is present. Chrome 150 H.264 playback remains unprobed.

  BAS ownership evidence: `playwright-driver/src/session/browser-manager.ts` launches Playwright Chromium; `playwright-driver/src/config.ts` reads `BROWSER_EXECUTABLE_PATH` (the docs' `PLAYWRIGHT_CHROMIUM_PATH` name does not match the runtime). `service.json` does not pass that variable, and BAS runtime status showed it empty/non-editable/restart-only on both existing instances. Capture CLI has no browser selector. The supported browser-cache command `cd scenarios/browser-automation-studio/playwright-driver && pnpm run browsers:ensure` restores the pinned Playwright browser, not an H.264-qualified browser. There is no persistent manifest/runtime override, but an invocation-scoped inherited environment can target a fresh named instance through direct `vrooli scenario start`, as traced below; no shared BAS configuration change is needed.

  Read-only instance snapshot: live healthy (API 17116, driver 24485, metrics 24478, UI 21794); shadow healthy (API 15372, driver 24465, metrics 24489, UI 21945). Both reported zero active automation sessions. Execution history was bounded to the latest 100 entries per instance and contained no active execution; this does not establish that no idle UI viewers exist. A browser-150 codec probe through an owner-supported isolated BAS configuration remains pending.

- 2026-10-20: BAS Chrome-150 playback-path research (read-only; no browser
  launch, BAS instance, config change, or lifecycle action). A fresh named BAS
  instance can receive `BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome` through
  the supported direct CLI lifecycle path. `vrooli scenario start` parses
  `--instance` in `internal/cli/scenariocli/commands.go`; the start handler
  constructs the local scenario service/lifecycle runner
  (`internal/cli/scenariohandlers/handlers.go` and
  `internal/cli/vroolicli/app.go`). Lifecycle child environments are derived
  from the runner process's `os.Environ()` in
  `internal/lifecycle/phases.go` via `envkit.ForeignScenario`, which retains
  ordinary variables such as `BROWSER_EXECUTABLE_PATH` while filtering
  caller/session identity variables. BAS `playwright-driver` declares only
  `METRICS_PORT` in `.vrooli/service.json`; its `playwright-driver/src/config.ts`
  reads `BROWSER_EXECUTABLE_PATH`, and `src/session/browser-manager.ts` passes
  it as Playwright Chromium's `executablePath`. The env-prefixed direct command
  is therefore the relevant supported path; a separate `StartScenario` control
  plane RPC has no arbitrary environment field and should not be used to infer
  caller-environment propagation. The custom browser path need not be added to
  shared service config.

  Bounded follow-up, when separately scheduled: start only a fresh instance,
  e.g. `BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome vrooli scenario start
  browser-automation-studio --instance h264-probe --json`, then address that
  instance explicitly with BAS CLI `--api-base` set to its returned API port.
  Confirm the driver reports Chrome 150, evaluate `canPlayType` for the exact
  exported MP4, then load that same content in a video element and assert
  `play()` succeeds and decoded playback advances (a capability string alone
  is insufficient). Retain browser version, codec result, source/output hash,
  and observed playback state. Do not modify live or shadow instances. Let the
  BAS capture execution close its own session; verify its session inventory is
  empty, and if a session remains use the BAS driver's lease-bound
  `POST /session/:id/close` with the exact execution and lease IDs before
  stopping the isolated instance with
  `vrooli scenario stop browser-automation-studio --instance h264-probe`.
  Verify stopped status for that same instance. Chrome 150 playback remains
  untested; this note establishes only the supported environment path.

- 2026-10-20: isolated Chrome 150 MP4 playback proof completed under
  `evidence/E2V/p8-matrix/browser-playback/`. Started only the previously
  stopped named BAS instance `h264-probe` with the inherited
  `BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome` variable. Lifecycle start
  operation `startop-7e007c136d1da081cf904b7523241e2e` reached healthy; BAS
  driver logged executable `/usr/bin/google-chrome` and actual version
  `150.0.7871.128`. The fresh routed Device Studio MP4 for job
  `78ab8ef8-3e51-49a7-8985-81fe763a304a` matched SHA-256
  `08e819f24f7ac625dfa3541e9864224e293bfe5dd17be28603b468c050a9ee8c`, size
  697,662 bytes, MIME `video/mp4`, 1920×1080, duration 2 seconds. Browser
  `canPlayType('video/mp4')` returned `maybe`; actual playback reported 3
  decoded frames, media time 0.066667 seconds, and advanced currentTime to
  0.323675 seconds while remaining unpaused. BAS applies anti-detection
  user-agent spoofing, so `navigator.userAgent` reports Chrome/120; browser
  identity is established by the driver launch event instead. The successful
  capture execution is `ccd9c937-bbd3-40dc-a8f9-89913253f6a8`; its exact flow,
  timeline, screenshot, result, sanitized launch events, and SHA manifest are
  retained in `p8-matrix/browser-playback/chrome150-success/`. The capture
  automatically closed its session; owner session inventory then reported
  zero active sessions. The isolated `h264-probe` instance was stopped and
  checked stopped. No shared BAS instance or Device Studio state was changed.
