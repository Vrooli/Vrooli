import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

(function bootstrapVisitedTrackerBridge() {
  if (typeof window === 'undefined' || window.parent === window || window.__visitedTrackerBridgeInitialized) {
    return;
  }

  let parentOrigin;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[VisitedTracker] Unable to determine parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'visited-tracker' });
  window.__visitedTrackerBridgeInitialized = true;
})();
