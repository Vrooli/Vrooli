# Preview Workspace Stable Perf Playbook

Use this when changing iframe loading, pane state, workspace layout, or scenario/resource picker behavior.

## Setup

Start App Monitor through its scenario lifecycle. The capture command must be
invoked inside an Agent Manager run that has a persisted, managed/fresh BAS
profile for the fixed `http://localhost:20000/apps/workspace` route. The script
does not select a permission profile or accept a browser executable/profile.
Direct shell runs without the Agent Manager identity and browser grant fail
closed before opening a session.

```bash
cd scenarios/app-monitor
make start
```

For React profiler marks, run a profile UI build before starting the scenario if the app is not already serving one:

```bash
cd scenarios/app-monitor/ui
pnpm build:profile
```

## Capture

Run the stable 8-pane workspace capture:

```bash
node scenarios/app-monitor/tools/perf/preview-workspace-capture.mjs \
  --panes 8 \
  --out /tmp/app-monitor/perf/preview-workspace-8pane.json \
  --trace /tmp/app-monitor/perf/preview-workspace-8pane.trace.json
```

The script opens one BAS-owned fresh session, adds panes through the App Monitor
workspace route, then records fixed `begin`, `after_mount`, `after_scroll`,
`after_picker` and `finish` phases. BAS enforces the persisted metric/trace
grants, fixed URL scope, phase order, capture/session binding, and the trace
duration/byte ceilings. The driver returns only five scalar metrics and, when
`--trace` is supplied and the run grants it, a bounded sanitized trace JSON.
The JSON contains allowlisted event fields; page arguments and arbitrary event
names are removed. `--trace` is a local output path and is never sent to BAS.

For a heavier ceiling run:

```bash
node scenarios/app-monitor/tools/perf/preview-workspace-capture.mjs \
  --panes 12 \
  --out /tmp/app-monitor/perf/preview-workspace-12pane.json
```

## What To Compare

- `counters.afterMount.iframes`: focused/pinned panes should load eagerly; offscreen panes should stay deferred.
- `counters.afterScroll.iframes`: should increase only after lower panes enter the viewport.
- `counters.afterPicker.tabCards`: should stay bounded until the user presses Show more.
- `counters.*.storageBytes`: should remain small and stable; it should not grow with bridge history.
- `errors`: `null` means console exception text is unknown through the typed
  Agent Access contract; do not infer that the browser had no console errors.
- `capture.taskRevision` and `capture.oracleRevision`: baseline identity for
  the fixed task and metric reader. Keep both in comparisons.

The UI now creates pane state through its supported workspace route. The
result preserves pane/iframe/card/profiler/storage counter meanings, but the
newly generated pane IDs and normalized layout are not byte-identical to the
former localStorage seed. Fixture tests are not live browser evidence. No
browser, target, or real grant is qualified until MB7 admits and validates a
run.

## Cleanup

If you built profile assets and need to restore the normal production bundle:

```bash
cd scenarios/app-monitor/ui
pnpm build
```
