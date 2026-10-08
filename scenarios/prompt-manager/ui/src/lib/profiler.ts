import type { ProfilerOnRenderCallback } from 'react'

export const onProfilerRender: ProfilerOnRenderCallback = (
  id,
  phase,
  actualDuration,
) => {
  try {
    performance.measure(`⚛ ${id} (${phase})`, {
      start: performance.now() - actualDuration,
      duration: actualDuration,
    })
  } catch {
    // Profiling must never affect the running app.
  }
}
