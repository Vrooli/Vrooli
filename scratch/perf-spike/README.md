# perf-spike

One-off Playwright + CDP script that captures a Chrome performance trace
while exercising swarm-manager. Validates whether the Tracing.start /
Tracing.end + IO.read pattern produces a useful artifact for component-level
React perf diagnosis on a stock production build.

If this proves out, fold the helper into BAS as
`browser-automation-studio perf trace …`.

## Run against the regular prod build

```bash
# swarm-manager must be running (vrooli scenario start swarm-manager)
node scratch/perf-spike/capture.js \
  --url http://localhost:21234 \
  --out /tmp/swarm-trace.json
```

Open `/tmp/swarm-trace.json` in Chrome → DevTools → Performance → "Load
profile…". The web-vitals sidecar lands at `/tmp/swarm-trace.web-vitals.json`.

## Run against the perf build (recommended for component-level diagnosis)

The perf build aliases `react-dom/client` → `react-dom/profiling` and keeps
function names through minification (see `ui/vite.config.ts`). It is *still a
production build* — minified, batched, no StrictMode double-renders — only
the React profiling instrumentation differs.

```bash
# 1. Tell the lifecycle to use the perf build on the next setup pass.
#    The build-ui step in .vrooli/service.json honours this env var.
export VROOLI_BUILD_MODE=profile

# 2. Restart so setup re-runs and produces the perf bundle.
vrooli scenario restart swarm-manager

# 3. Capture the trace.
node scratch/perf-spike/capture.js \
  --url http://localhost:21234 \
  --out /tmp/swarm-trace-profile.json

# 4. When done auditing, restore the regular bundle.
unset VROOLI_BUILD_MODE
vrooli scenario restart swarm-manager
```

> **Why an env var?** `vrooli scenario restart` always runs the setup phase
> with `force_setup=true`, which re-runs the UI build. The lifecycle builder
> reads `VROOLI_BUILD_MODE` and selects the `build:profile` package script for
> the profile channel, so the choice is plain argv rather than a conditional
> inside the script. The env var survives across `restart` cycles and
> consistently produces a perf bundle for the lifetime of the shell that set
> it. Unset it and re-restart to revert.

## What the script scripts

1. Wait for app shell to render
2. Resize the sidebar handle back-and-forth (~3 s) on the graph page
3. Switch to the "backlog" tab
4. Type into the sidebar search to trigger filter pipeline
5. Navigate to /command-post
6. Resize the sidebar handle back-and-forth (~3 s) on Command Post
7. Stop tracing

The tab/search/resize sequence is chosen to surface the four interactions the
user reported as laggy.

## Borrowed Playwright

`capture.js` requires `rebrowser-playwright` from BAS's
`playwright-driver/node_modules`. No fresh install needed. If/when this
graduates into BAS as a real CLI command, that path becomes a normal
package dep.
