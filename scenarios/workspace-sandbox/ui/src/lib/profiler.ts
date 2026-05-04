/**
 * Shared <React.Profiler> onRender callback.
 *
 * Wrapping a subtree in <Profiler id="X" onRender={onProfilerRender}> emits a
 * `performance.measure` entry every time that subtree commits. Those entries
 * appear under user_timing in Chrome DevTools' Performance panel and in any
 * trace JSON captured via CDP `Tracing.start`. This is component-level commit
 * timing without needing the React DevTools extension attached.
 *
 * Why it's safe to ship permanently: <React.Profiler>'s `onRender` is only
 * invoked when React's profiling instrumentation is present. The standard
 * production `react-dom` bundle strips that instrumentation, so the callback
 * is never called in a regular `pnpm run build`. The fiber for <Profiler>
 * still exists, but it costs effectively nothing at runtime. Only the
 * perf-build channel (`pnpm run build:profile`, or
 * `VROOLI_BUILD_MODE=profile pnpm run build` — see vite.config.ts) keeps the
 * instrumentation in. Wrappers are inert in default prod, fire onRender in
 * the perf bundle.
 */

import type { ProfilerOnRenderCallback } from "react";

export const onProfilerRender: ProfilerOnRenderCallback = (
  id,
  phase,
  actualDuration,
) => {
  try {
    performance.measure(`⚛ ${id} (${phase})`, {
      start: performance.now() - actualDuration,
      duration: actualDuration,
    });
  } catch {
    // Some older browsers reject the object form of performance.measure().
    // A measurement failure must never surface in the running app.
  }
};
