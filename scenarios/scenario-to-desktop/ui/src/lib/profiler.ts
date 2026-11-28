import type { ProfilerOnRenderCallback } from "react";

// The production React profiling build calls this boundary callback. Keeping
// measurements in the browser Performance Timeline makes profile captures
// portable: local desktop packs and future bridge-hosted packs can collect the
// same standard browser evidence without a scenario-specific transport.
export const onProfilerRender: ProfilerOnRenderCallback = (
  id,
  phase,
  actualDuration,
  baseDuration,
  startTime,
  commitTime,
) => {
  if (typeof performance.measure !== "function") {
    return;
  }

  performance.measure(`⚛ ${id} ${phase}`, {
    start: startTime,
    end: commitTime,
    detail: { actualDuration, baseDuration },
  });
};
