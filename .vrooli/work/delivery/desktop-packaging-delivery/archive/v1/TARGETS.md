# Validation targets

Tiers follow `large-effort-orchestration` §5. The orchestrator changes a tier
when new evidence arrives. An unavailable target never blocks the goal.

| Target | Tier | How |
|---|---|---|
| Linux x64, this host (Ubuntu 24.04, 32 cores, 60 GiB, NVIDIA RTX 4070 Ti SUPER) | required | Xvfb automation runs plus one real-display run per experiment cell when a desktop session exists. Every number reported here comes from this host unless labelled otherwise. |
| macOS x64 (`minimouse`, Intel Mac mini, see `docs/architecture/remote-desktop-control.md`) | occasional | Use the Vrooli bridge when it works. WKWebView automation is missing, so macOS evidence is launch and resource metrics only. Log friction in WORKAROUNDS.md. |
| Windows x64 | unavailable | No machine is connected. Wine and compile-only builds do not count as evidence. |
| macOS arm64 | unavailable | No machine is connected. |

## Reference scenarios (experiment subjects)

| Scenario | Why | Stack class |
|---|---|---|
| `hello-desktop` | Canonical ramp fixture with an existing baseline | go_module API + static UI bundle |
| `system-monitor` | Replaces secrets-manager per DPF-FB-014: secrets-manager required bundled Postgres, secret-bearing export and credential issuance, and its current desktop journey failed on an unchanged pointer surface. system-monitor has optional dependencies, needs no database/secrets bundle, and is already live; it has no prior desktop record, so E1 must prove its first current-source build. | go_module API + pnpm_vite UI |
| `web-console` | Current bundled reference: pipeline `9be28246-c89b-59c9-9078-8ce1ef17f568` and its app-specific journey produced comparable 10-run cold and warm cohorts, all required metrics, and 0% cold page-cache residency. The earlier proxy and isolated-target journeys failed; the isolated target still reports an unrelated UI freshness decoding error. | go_module API + pnpm_vite UI |
| `landing-page-business-suite` | D2-approved substitute for `git-control-tower`: it has comparable 10-run cold and warm cohorts with passing journeys and all required metrics. Earlier captures were invalidated by stale proxy/UI state and a short measured hold; the current baseline uses the rebuilt served bundle and extended hold. Idle CPU p50 is high (91.4% cold, 89.6% warm) and host pressure rose during both batches, so these measurements remain qualified with that caveat. | go_module API + pnpm_vite UI |
| `git-control-tower` | Excluded and replaced by the D2-approved LPBS substitute. Its current proxy artifact emitted metrics, but the `desktop.launch.baseline` journey failed at `pointer_click` (`surface_unchanged`); the current bundled attempt panicked because the API could not find `../.vrooli/service.json`. No source change was made. | go_module API + pnpm_vite UI |

E1 confirms each subject builds and runs on the baseline strategy. If one
cannot, E1 replaces it with the next scenario that has a desktop record (from
`browser-automation-studio`, `vrooli-onboarding`,
`landing-page-business-suite`) and records the substitution here.

For system-monitor analysis, its own host-metrics polling contributes real
application work to `idle_cpu_pct`; record this caveat in
`docs/internal/packaging-baseline.md`. Cross-strategy comparisons remain valid.

## Running scenarios during epochs

- `scenario-to-desktop`, `performance-health`, `code-facts`,
  `deployment-manager` and `program-runtime` are shared live services. Restart
  one only through `vrooli scenario restart <name>`, after checking that
  `test-genie runs list --scenario <name>` shows no in-flight validation of
  it. Record each
  restart in the epoch slice log.
- Live BAS showed `start-failed` at commissioning (2026-10-06). The BAS
  delivery team runs a shadow (`bas-goal`). Journey parity that needs BAS falls
  back to the scenario-to-desktop xdotool platform journey plus a CDP
  screenshot check. Record the fallback in WORKAROUNDS.md, and never restart
  BAS from this goal.

## Daily qualification

Once a day, the orchestrator runs the packaged-launch capture defined in E1
against `hello-desktop` with the current default strategy, at 5 cold and
5 warm runs. It uses the live instances: this goal has no shadow, because the
artifact under test is a built package rather than a running scenario. It keeps the JSON at
`~/.vrooli/evidence/desktop-packaging/qualification-<date>.json` (outside the
repo) and records p50 cold `app_interactive` and steady PSS in the active
epoch's log. At E1 acceptance, the orchestrator replaces this paragraph with
the exact command.

Until E1 lands, the qualification is:

    scenario-to-desktop validation run --scenario hello-desktop \
      --artifact-digest <sha256:...> --artifact-path <AppImage> \
      --platform linux --journey <id> --profile <name>

Run it with the current artifact. Flags and journey or profile IDs are in
`scenarios/scenario-to-desktop/docs/reference/cli-commands.md`
§Agent-driven validation. If the result is worse than the previous day by more than 15%,
or the run fails, the orchestrator adds a Qualification findings item to
QUEUE.md. It never reopens an epoch for this.

## Quality bands (destination)

Measured on Linux x64 with `packaged-launch.v1` (design §6), nearest-rank over
at least 10 runs. These are targets for the best eligible strategy for each
reference scenario. The baseline strategy only has to be measured honestly.

- Cold time to interactive: hello-desktop p50 ≤ 1.0 s and p95 ≤ 1.5 s. Real
  reference scenarios p50 ≤ 2.0 s.
- Warm time to interactive: p50 ≤ 0.6 s for hello-desktop, ≤ 1.2 s for the real
  scenarios.
- First window (splash or main): p95 ≤ 0.5 s for every strategy.
- Steady memory (PSS, whole tree): at least 30% below the measured
  `electron-sidecar` baseline for the same scenario, or a recorded explanation
  of why the floor is set by the app rather than the packaging.
- Idle CPU: < 1% of one core, averaged over 60 s after interactive.
- Artifact size: reported for every strategy. No band; it is a selection
  input.
- Parity: the scenario's journey passes on the selected strategy. A strategy
  without parity evidence is never auto-selected.
- Reliability: ≥ 99% of launches reach `app_interactive` in a 100-launch soak
  for any strategy that becomes a default.

## Unverified until a sensor exists

- Windows and macOS arm64 numbers.
- Real-display GPU effects on a host without a desktop session.
- Update-size savings in production channels.
