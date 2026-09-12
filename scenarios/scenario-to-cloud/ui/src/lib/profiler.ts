import type { ProfilerOnRenderCallback } from "react";

// The callback is inert in the regular production React bundle and becomes
// observable only in the explicit profile build channel.
export const onProfilerRender: ProfilerOnRenderCallback = (id, phase, actualDuration) => {
  try {
    performance.measure(`⚛ ${id} (${phase})`, {
      start: performance.now() - actualDuration,
      duration: actualDuration,
    });
  } catch {
    // Measurement must never affect the running application.
  }
};
