/**
 * Shared <React.Profiler> onRender callback (managed by Performance Health).
 *
 * Wrapping a subtree in <Profiler id="X" onRender={onProfilerRender}> emits a
 * performance.measure entry every time that subtree commits. The "⚛" prefix
 * groups them in Chrome DevTools' Performance panel. onRender only fires when
 * React's profiling instrumentation is present (the perf-build channel), so the
 * wrapper is inert in regular prod and safe to ship permanently.
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
    // Never let a measurement failure surface in the running app.
  }
};
