import type { ProfilerOnRenderCallback } from "react";

// React invokes this only in the dedicated profiling build, where the Vite
// aliases preserve React's profiling instrumentation. The normal production
// bundle keeps the boundary inert while retaining a stable trace seam.
export const onProfilerRender: ProfilerOnRenderCallback = (id, phase, actualDuration) => {
	try {
		performance.measure(`⚛ ${id} (${phase})`, {
			start: performance.now() - actualDuration,
			duration: actualDuration
		});
	} catch {
		// Measurements are diagnostic only and must never affect application flow.
	}
};
