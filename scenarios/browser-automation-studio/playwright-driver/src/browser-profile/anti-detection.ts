/**
 * Anti-Detection Module
 *
 * Provides browser launch args and init scripts for bypassing bot detection.
 *
 * ARCHITECTURE:
 * - Launch args: Built inline (simple conditionals)
 * - Init scripts: Composed from individual patches in ./patches.ts
 *
 * CONTROL SURFACE: See ./config.ts for default values and their tradeoffs.
 * The defaults are designed to make automated browsers appear as legitimate
 * user browsers, reducing detection by anti-bot systems.
 *
 * CHANGE AXIS: New Anti-Detection Patch
 * Add new patches in ./patches.ts, not here. This module orchestrates.
 * See patches.ts for the full list of available patches and how to add new ones.
 */

import type { BrowserContext } from 'rebrowser-playwright';
import type { AntiDetectionSettings, FingerprintSettings } from '../types/browser-profile';
import { composePatches } from './patches';

// Re-export for testing and debugging
export { getEnabledPatches } from './patches';

/**
 * Build Chromium launch arguments for anti-detection.
 *
 * DECISION: Launch args vs init scripts
 * - Launch args: Apply before any page loads, affect all contexts
 * - Init scripts: Run in page context, can be more targeted
 *
 * These args disable Chrome features that reveal automation.
 */
export function buildAntiDetectionArgs(settings: AntiDetectionSettings): string[] {
  const args: string[] = [];

  // DECISION: Disable AutomationControlled blink feature
  // This removes the "Chrome is being controlled by automated test software" infobar
  // and prevents detection via navigator.webdriver in some cases
  if (settings.disable_automation_controlled) {
    args.push('--disable-blink-features=AutomationControlled');
  }

  // DECISION: Disable WebRTC to prevent IP leakage
  // WebRTC can reveal real IP even behind VPN/proxy
  if (settings.disable_webrtc) {
    args.push('--disable-webrtc');
    args.push('--disable-webrtc-encryption');
    args.push('--disable-webrtc-hw-decoding');
    args.push('--disable-webrtc-hw-encoding');
  }

  return args;
}

/**
 * Generate the init script content for anti-detection patches.
 *
 * DECISION: Compose patches at runtime
 * Patches are composed from individual generators in ./patches.ts.
 * This allows:
 * - Each patch to be tested independently
 * - Easy addition of new patches without modifying this function
 * - Clear visibility into which patches are applied
 *
 * @see patches.ts for the patch registry and individual patch implementations
 */
export function generateAntiDetectionScript(
  antiDetection: AntiDetectionSettings,
  fingerprint: FingerprintSettings
): string {
  return composePatches(antiDetection, fingerprint);
}

/**
 * Apply anti-detection patches to a browser context.
 */
export async function applyAntiDetection(
  context: BrowserContext,
  antiDetection: AntiDetectionSettings,
  fingerprint: FingerprintSettings
): Promise<void> {
  const script = generateAntiDetectionScript(antiDetection, fingerprint);

  if (script) {
    await context.addInitScript({ content: script });
  }
}
