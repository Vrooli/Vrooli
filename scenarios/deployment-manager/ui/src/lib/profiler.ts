import type { ProfilerOnRenderCallback } from "react";

export const onProfilerRender: ProfilerOnRenderCallback = (
  id,
  phase,
  actualDuration,
  baseDuration,
  startTime,
  commitTime,
) => {
  if (typeof performance === "undefined") return;
  const measureName = `⚛ ${id} ${phase}`;
  try {
    performance.measure(measureName, { start: startTime, end: commitTime });
  } catch {
    // Browsers without the object-form PerformanceMeasureOptions are still
    // supported; the numeric telemetry remains useful in the console.
    performance.measure(measureName);
  }
  if (import.meta.env.DEV) {
    console.debug("react_profiler", { id, phase, actualDuration, baseDuration, startTime, commitTime });
  }
};
