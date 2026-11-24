/**
 * Playwright Provider Implementation
 *
 * Configures the active playwright provider for the application.
 * Uses rebrowser-playwright by default for its browser APIs and selected
 * signal-adjustment code paths. No detection result is claimed.
 *
 * ## Environment Configuration
 *
 * Set PLAYWRIGHT_PROVIDER environment variable to switch providers:
 * - PLAYWRIGHT_PROVIDER=rebrowser-playwright (default) - Selected signal patches, isolated context
 * - PLAYWRIGHT_PROVIDER=playwright - Standard behavior, main context
 *
 * ## Recording Implications by Provider
 *
 * ### rebrowser-playwright (default)
 * - page.evaluate() runs in ISOLATED context (invisible to page)
 * - exposeBinding() only works from ISOLATED context
 * - REQUIRES: HTML injection via route interception for recording script
 * - REQUIRES: fetch() to intercepted URL for event communication
 * - Detection behavior is unqualified without a dated per-site corpus
 *
 * ### playwright (alternative)
 * - page.evaluate() runs in MAIN context
 * - exposeBinding() works from any context
 * - SIMPLER: addInitScript() works directly for recording
 * - SIMPLER: exposeBinding() works for event communication
 * - Detection behavior is unqualified without a dated per-site corpus
 *
 * ## When to Switch Providers
 *
 * Switch to 'playwright' temporarily if:
 * 1. You're debugging recording issues and need simpler execution model
 * 2. You're testing in controlled environments that don't block automation
 * 3. You need Firefox/WebKit support (rebrowser-playwright is Chromium-only)
 *
 * Choose a provider for API compatibility and tested task behavior. Qualify
 * detection separately for each supported mode and site.
 *
 * @module playwright/provider
 */

import { chromium } from 'rebrowser-playwright';
import { logger } from '../utils';
import type { PlaywrightProvider, PlaywrightProviderName, PlaywrightCapabilities } from './types';

/**
 * Default provider name when PLAYWRIGHT_PROVIDER env var is not set.
 */
const DEFAULT_PROVIDER: PlaywrightProviderName = 'rebrowser-playwright';

/**
 * Get the configured provider name from environment.
 *
 * @returns The provider name to use
 */
export function getConfiguredProviderName(): PlaywrightProviderName {
  const envProvider = process.env.PLAYWRIGHT_PROVIDER;
  if (envProvider === 'playwright' || envProvider === 'rebrowser-playwright') {
    return envProvider;
  }
  return DEFAULT_PROVIDER;
}

/**
 * Capability definitions for each provider.
 *
 * These document the behavioral differences that affect recording architecture.
 * When adding a new provider, add its capabilities here.
 */
export const PROVIDER_CAPABILITIES: Record<PlaywrightProviderName, PlaywrightCapabilities> = {
  'rebrowser-playwright': {
    /**
     * rebrowser-playwright runs page.evaluate() in ISOLATED context. This
     * driver behavior does not qualify any site-specific detection outcome.
     * It means:
     * - Scripts can't wrap History API in the page
     * - Scripts can't access page's global variables
     * - Scripts are invisible to page's JavaScript
     *
     * Workaround: Inject scripts via HTML modification (route interception)
     */
    evaluateIsolated: true,

    /**
     * exposeBinding() callbacks only fire when called from ISOLATED context
     * (i.e., from page.evaluate()). Scripts in MAIN context (injected HTML)
     * cannot trigger bindings directly.
     *
     * Workaround: Use fetch() to a route-intercepted URL for event communication
     */
    exposeBindingIsolated: true,

    /**
     * rebrowser-playwright includes patches that adjust selected browser
     * signals. This flag reports the presence of those mechanisms only; it
     * does not report detection results or production-site compatibility.
     */
    hasAntiDetection: true,
  },

  playwright: {
    /**
     * Standard Playwright runs page.evaluate() in the MAIN context. That
     * affects page-global access; this alone does not establish a detection
     * outcome.
     */
    evaluateIsolated: false,

    /**
     * Standard playwright's exposeBinding() works from any context.
     * Scripts injected via addInitScript can call bindings directly.
     */
    exposeBindingIsolated: false,

    /**
     * Standard Playwright does not include BAS's custom signal-adjustment
     * patches. Its site-specific detection outcome is not qualified here.
     */
    hasAntiDetection: false,
  },
};

/**
 * Create a playwright provider with the given name.
 *
 * NOTE: Currently only rebrowser-playwright is installed as a dependency.
 * To use 'playwright', you must:
 * 1. Install: `pnpm add playwright`
 * 2. The dynamic import below will then work
 *
 * For now, this function always returns the rebrowser-playwright provider
 * but logs a warning if a different provider was requested.
 *
 * @param name - Provider name (defaults to environment config or 'rebrowser-playwright')
 * @returns PlaywrightProvider instance
 */
export function createPlaywrightProvider(name?: PlaywrightProviderName): PlaywrightProvider {
  const providerName = name ?? getConfiguredProviderName();
  const capabilities = PROVIDER_CAPABILITIES[providerName];

  // Currently we only have rebrowser-playwright installed
  // Log warning if different provider requested but not available
  if (providerName === 'playwright') {
    logger.warn(
      '[playwright-provider] Standard playwright requested but only rebrowser-playwright is installed. ' +
        'To use standard playwright: pnpm add playwright. ' +
        'Using rebrowser-playwright instead.'
    );
    return {
      name: 'rebrowser-playwright',
      chromium,
      capabilities: PROVIDER_CAPABILITIES['rebrowser-playwright'],
    };
  }

  return {
    name: providerName,
    chromium,
    capabilities,
  };
}

/**
 * Active playwright provider instance.
 *
 * Uses rebrowser-playwright by default for its documented driver behavior and
 * selected signal adjustments; site-specific detection outcomes are unqualified.
 * The capabilities document the behavioral differences that affect
 * how the recording system must be implemented.
 *
 * To switch providers at runtime, use createPlaywrightProvider() instead.
 *
 * @example
 * ```typescript
 * import { playwrightProvider } from './playwright';
 *
 * // Launch browser using the provider
 * const browser = await playwrightProvider.chromium.launch({
 *   headless: true,
 * });
 *
 * // Check if we need special handling for isolated context
 * if (playwrightProvider.capabilities.evaluateIsolated) {
 *   // Use HTML injection for scripts
 * }
 * ```
 */
export const playwrightProvider: PlaywrightProvider = createPlaywrightProvider();

/**
 * Log the current provider configuration.
 *
 * Useful for debugging to confirm which provider is active.
 */
export function logProviderConfig(): void {
  const provider = playwrightProvider;
  logger.info('[playwright-provider] Active configuration:', {
    name: provider.name,
    evaluateIsolated: provider.capabilities.evaluateIsolated,
    exposeBindingIsolated: provider.capabilities.exposeBindingIsolated,
    hasAntiDetection: provider.capabilities.hasAntiDetection,
    implications: {
      scriptInjection: provider.capabilities.evaluateIsolated
        ? 'HTML injection via route interception'
        : 'addInitScript() or page.evaluate()',
      eventCommunication: provider.capabilities.exposeBindingIsolated
        ? 'fetch() to route-intercepted URL'
        : 'exposeBinding() directly',
      botDetection: 'Site-specific detection outcome is unqualified',
    },
  });
}
