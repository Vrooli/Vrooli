/**
 * Recording Cleanup Script
 *
 * This script is injected to deactivate recording mode.
 * It simply sets a flag that prevents further event capture.
 *
 * @see ../injector.ts - The Node.js module that injects this script
 */

(function () {
  window.__recordingActive = false;
  console.log('[Recording] Deactivated');
})();
