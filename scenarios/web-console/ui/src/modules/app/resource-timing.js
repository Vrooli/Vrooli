export function initializeResourceTimingGuard() {
  if (typeof performance === "undefined") {
    return;
  }
  if (typeof performance.setResourceTimingBufferSize === "function") {
    try {
      performance.setResourceTimingBufferSize(1);
    } catch (error) {
      console.warn("Unable to adjust resource timing buffer size:", error);
    }
  }
  if (typeof performance.clearResourceTimings === "function") {
    try {
      performance.clearResourceTimings();
    } catch (error) {
      console.warn("Unable to clear initial resource timings:", error);
    }
  }
}
