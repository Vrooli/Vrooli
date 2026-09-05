import type { ProfilerOnRenderCallback } from "react";

// AI_CHECK: REACT_PERFORMANCE=1 | LAST: 2026-05-04
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
    // Profiling must never affect application behaviour.
  }
};
