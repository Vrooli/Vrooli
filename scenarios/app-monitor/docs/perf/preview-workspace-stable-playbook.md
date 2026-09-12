# Preview Workspace Stable Perf Playbook

Use this when changing iframe loading, pane state, workspace layout, or scenario/resource picker behavior.

## Setup

Start app-monitor through the scenario lifecycle:

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
  --url http://localhost:20000 \
  --panes 8 \
  --out /tmp/app-monitor/perf/preview-workspace-8pane.json \
  --trace /tmp/app-monitor/perf/preview-workspace-8pane.zip
```

The script launches Chrome through the DevTools Protocol. Set `CHROME_BIN=/path/to/chrome` or pass `--chrome /path/to/chrome` if `google-chrome` is not on `PATH`.

For a heavier ceiling run:

```bash
node scenarios/app-monitor/tools/perf/preview-workspace-capture.mjs \
  --url http://localhost:20000 \
  --panes 12 \
  --out /tmp/app-monitor/perf/preview-workspace-12pane.json
```

## What To Compare

- `counters.afterMount.iframes`: focused/pinned panes should load eagerly; offscreen panes should stay deferred.
- `counters.afterScroll.iframes`: should increase only after lower panes enter the viewport.
- `counters.afterPicker.tabCards`: should stay bounded until the user presses Show more.
- `counters.*.storageBytes`: should remain small and stable; it should not grow with bridge history.
- `errors`: must not contain `QuotaExceededError`.

## Cleanup

If you built profile assets and need to restore the normal production bundle:

```bash
cd scenarios/app-monitor/ui
pnpm build
```
