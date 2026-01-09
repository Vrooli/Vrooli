import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_STATE_KEY = '__nutritionTrackerBridgeInitialized' as const;
let initialized = false;

declare global {
  interface Window {
    __nutritionTrackerBridgeInitialized?: boolean;
  }
}

export function ensureIframeBridge() {
  if (typeof window === 'undefined') {
    return;
  }

  if (initialized || window.parent === window || window[BRIDGE_STATE_KEY]) {
    initialized = true;
    return;
  }

  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[NutritionTracker] Unable to determine parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'nutrition-tracker' });
  window[BRIDGE_STATE_KEY] = true;
  initialized = true;
}
