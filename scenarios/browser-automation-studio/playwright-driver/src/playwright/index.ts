/**
 * Playwright Provider Module
 *
 * Provides a configurable playwright implementation for the application.
 * Use this instead of importing from 'rebrowser-playwright' directly
 * to enable future provider switching.
 *
 * ## Quick Start
 *
 * ```typescript
 * import { playwrightProvider } from './playwright';
 *
 * // Launch browser using the provider
 * const browser = await playwrightProvider.chromium.launch();
 *
 * // Check capabilities for conditional logic
 * if (playwrightProvider.capabilities.evaluateIsolated) {
 *   // Use HTML injection for scripts (rebrowser-playwright)
 * } else {
 *   // Use addInitScript directly (standard playwright)
 * }
 * ```
 *
 * ## Environment Configuration
 *
 * Set PLAYWRIGHT_PROVIDER to switch providers:
 * - PLAYWRIGHT_PROVIDER=rebrowser-playwright (default)
 * - PLAYWRIGHT_PROVIDER=playwright
 *
 * ## Troubleshooting Recording Issues
 *
 * Import RECORDING_TROUBLESHOOTING for common issue diagnosis:
 * ```typescript
 * import { RECORDING_TROUBLESHOOTING } from './playwright';
 * console.log(RECORDING_TROUBLESHOOTING.noEvents.checkList);
 * ```
 *
 * @module playwright
 */

// Provider instance and factory
export {
  playwrightProvider,
  createPlaywrightProvider,
  getConfiguredProviderName,
  logProviderConfig,
  PROVIDER_CAPABILITIES,
} from './provider';

// Types
export type {
  PlaywrightProvider,
  PlaywrightCapabilities,
  PlaywrightProviderName,
  Browser,
  BrowserContext,
  Page,
} from './types';

// Troubleshooting guide
export { RECORDING_TROUBLESHOOTING } from './types';
