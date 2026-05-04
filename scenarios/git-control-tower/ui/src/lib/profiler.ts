/**
 * Shared <React.Profiler> onRender callback.
 *
 * Wrapping a subtree in <Profiler id="X" onRender={onProfilerRender}> emits a
 * `performance.measure` entry every time that subtree commits. Those entries
 * appear in Chrome DevTools' Performance panel (under user_timing) and in any
 * trace JSON captured via CDP `Tracing.start`.
 *
 * The callback only fires when React's profiling instrumentation is present —
 * i.e. when the perf-build channel (`pnpm run build:profile` or
 * `VROOLI_BUILD_MODE=profile pnpm run build`) is active. In a regular prod
 * build the wrapper is inert; safe to leave permanently at subtree boundaries.
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
  }
};
