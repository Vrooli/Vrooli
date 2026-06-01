# UI Performance Capture

This directory ships a starting-point capture script for running a headless,
reproducible Chrome performance trace against this scenario's UI. Use it
together with the `scenario-performance-audit` skill, which is the
authoritative methodology:

```bash
prompt-manager skill read scenario-performance-audit
```

## What's in here

- `capture.template.js` — Playwright + CDP tracing script. Includes
  reusable interaction helpers (`dragHorizontalOnce`, `findScrollableAncestor`)
  and a placeholder `exerciseTarget` function you customise per audit.

## Quick start (Phase 4 of the audit)

```bash
scenario_id=<this-scenario-id>
workdir="/tmp/${scenario_id}/perf"
mkdir -p "${workdir}"

# Copy the template and customise exerciseTarget for the audit.
cp ui/perf/capture.template.js "${workdir}/capture.js"
${EDITOR:-vi} "${workdir}/capture.js"

# Restart the scenario in profile-mode so the served bundle emits ⚛ entries.
vrooli scenario restart "${scenario_id}"

# Run the capture against the scenario's UI port.
port=$(vrooli scenario port "${scenario_id}" UI_PORT)
node "${workdir}/capture.js" "http://localhost:${port}" "${workdir}/trace.json"
```

The capture writes two files:

- `trace.json` — load in Chrome DevTools Performance panel for visual analysis.
- `trace.web-vitals.json` — long tasks, paint, LCP captured via
  `PerformanceObserver`.

After capture, follow Phase 5 (analyse) and Phase 7 (persist) of the skill
to produce a `docs/perf/<date>-<slug>.md` artifact validated by
`knowledge-observatory docs audit`.

## Why this lives in `ui/perf/` (not at the scenario root)

`ui/perf/` is sibling to `ui/scripts/` and `ui/test-utils/` — runtime tooling
specific to the UI. The capture script targets a running UI bundle, so it
belongs in the UI tree, not at the scenario root where it would imply
cross-stack relevance.

## Don't ship perf traces in git

`*.json` traces are 40+MB. They live in `/tmp/<scenario>/perf/` and are
referenced from the persisted `docs/perf/` doc by absolute path. The doc
survives; the trace is allowed to be GC'd from `/tmp`.
