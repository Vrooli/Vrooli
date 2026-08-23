/**
 * Shared <React.Profiler> onRender callback.
 *
 * What this is for
 * ────────────────
 * Wrapping a subtree in <Profiler id="X" onRender={onProfilerRender}> emits a
 * `performance.measure` entry every time that subtree commits. Those entries
 * appear in Chrome DevTools' Performance panel (under user_timing) and in any
 * trace JSON captured via CDP `Tracing.start`. That gives component-level
 * commit timing without needing the React DevTools extension attached.
 *
 * Why it's safe to ship permanently
 * ─────────────────────────────────
 * <React.Profiler>'s `onRender` is only invoked when React's profiling
 * instrumentation is present. The standard production `react-dom` bundle
 * strips that instrumentation, so the callback is **never called** in a
 * regular `pnpm run build`. The fiber for <Profiler> still exists, but it
 * costs effectively nothing — microseconds at mount, zero at runtime.
 *
 * Only the perf-build channel (`pnpm run build:profile`, or
 * `VROOLI_BUILD_MODE=profile vrooli scenario restart <name>`, which selects
 * that same script) keeps the instrumentation in. So:
 *
 *   - Default prod: <Profiler> wrappers are inert. No measurable impact.
 *   - Perf build:   wrappers fire onRender, this util emits user_timing.
 *
 * This means you can leave wrappers in source permanently at meaningful
 * subtree boundaries, and any future audit just builds the perf bundle to
 * get component-level signal — no code changes required.
 *
 * Cost in the perf build
 * ──────────────────────
 * ~5–10 % CPU per wrapped subtree per commit + one `performance.measure`
 * call. Acceptable for an audit; not what you'd want as the default ship.
 */

import type { ProfilerOnRenderCallback } from "react";

export const onProfilerRender: ProfilerOnRenderCallback = (
  id,
  phase,
  actualDuration,
) => {
  // performance.measure with explicit start/duration lands as a user_timing
  // entry exactly aligned to React's commit window. The "⚛" prefix groups
  // these together in the DevTools Performance panel timeline.
  try {
    performance.measure(`⚛ ${id} (${phase})`, {
      start: performance.now() - actualDuration,
      duration: actualDuration,
    });
  } catch {
    // Some older browsers reject the object form of performance.measure().
    // We never want a measurement failure to surface in the running app.
  }
};
