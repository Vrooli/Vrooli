import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

let initialized = false;

export function ensureIframeBridge() {
  if (typeof window === 'undefined') {
    return;
  }

  if (initialized || window.parent === window) {
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
  initialized = true;
}
