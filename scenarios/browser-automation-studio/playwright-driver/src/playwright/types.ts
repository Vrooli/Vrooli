/**
 * Playwright Provider Types
 *
 * Defines the interface for swappable Playwright implementations.
 * This abstraction allows switching between:
 * - rebrowser-playwright: Anti-detection patches, isolated context evaluation
 * - playwright: Standard behavior, main context evaluation
 *
 * ## Quick Decision Guide
 *
 * Use `rebrowser-playwright` when:
 * - Recording on production websites with bot detection
 * - Running automations that need to appear human
 * - Accessing protected sites that block standard Playwright
 *
 * Use `playwright` when:
 * - Debugging recording issues (simpler execution model)
 * - Testing in controlled environments
 * - Need Firefox/WebKit support
 * - Don't need to evade bot detection
 *
 * ## Environment Configuration
 *
 * Set PLAYWRIGHT_PROVIDER environment variable to switch:
 * - PLAYWRIGHT_PROVIDER=rebrowser-playwright (default)
 * - PLAYWRIGHT_PROVIDER=playwright
 *
 * @module playwright/types
 */

import type { Browser, BrowserContext, Page, chromium as ChromiumType } from 'rebrowser-playwright';

// Re-export common types for convenience
export type { Browser, BrowserContext, Page };

/**
 * Provider names for type safety
 */
export type PlaywrightProviderName = 'rebrowser-playwright' | 'playwright';

/**
 * Playwright Provider Capabilities
 *
 * Documents the behavioral differences between playwright implementations.
 * These flags help code adapt its behavior based on the active provider.
 *
 * ## Context Isolation Explained
 *
 * **MAIN Context**: The JavaScript world where the page's own scripts run.
 * - History API (pushState, replaceState) can be wrapped successfully
 * - Can access page's global variables
 * - Event listeners work as expected
 * - Detectable by bot detection scripts
 *
 * **ISOLATED Context**: A separate JavaScript world created by Playwright.
 * - Cannot access MAIN world variables directly
 * - History API wrapping doesn't affect the page
 * - Protects automation from page interference
 * - Harder for bot detection to find
 *
 * ## Why This Matters for Recording
 *
 * The recording script needs to:
 * 1. Wrap History API to capture navigation (requires MAIN context)
 * 2. Add event listeners for user actions (works in either context)
 * 3. Communicate events back to Node.js (method varies by context)
 *
 * With rebrowser-playwright (isolated):
 * - page.evaluate() runs in ISOLATED context
 * - exposeBinding() callbacks only fire from ISOLATED context
 * - Must inject script via HTML modification to run in MAIN
 * - Must use fetch() to route-intercepted URL for event communication
 *
 * With standard playwright (main):
 * - page.evaluate() runs in MAIN context
 * - exposeBinding() works directly
 * - addInitScript() runs in MAIN context
 * - Simpler architecture but detectable
 */
export interface PlaywrightCapabilities {
  /**
   * Whether page.evaluate() runs in an ISOLATED context.
   *
   * - true (rebrowser-playwright): Scripts run in isolated world, can't wrap History API
   * - false (playwright): Scripts run in MAIN world, can wrap everything
   *
   * If true, use HTML injection instead of evaluate() for scripts that need
   * to interact with the page's native APIs.
   */
  evaluateIsolated: boolean;

  /**
   * Whether exposeBinding() callbacks only work from ISOLATED context.
   *
   * - true (rebrowser-playwright): Binding only callable from page.evaluate()
   * - false (playwright): Binding callable from page's own scripts
   *
   * If true, use fetch() to a route-intercepted URL instead of bindings
   * for event communication from injected scripts.
   */
  exposeBindingIsolated: boolean;

  /**
   * Whether the provider includes anti-detection patches.
   *
   * - true (rebrowser-playwright): Hides automation markers, passes bot checks
   * - false (playwright): navigator.webdriver=true, easily detected
   *
   * Use rebrowser-playwright for production recording to avoid blocking.
   * Use standard playwright for testing (faster, simpler debugging).
   */
  hasAntiDetection: boolean;
}

/**
 * Playwright Provider Interface
 *
 * Abstracts the playwright implementation to allow dependency injection.
 * This enables:
 * 1. Switching providers without code changes
 * 2. Testing with standard playwright (simpler mocking)
 * 3. Future support for additional implementations
 *
 * ## Capability Comparison Matrix
 *
 * | Capability                | rebrowser-playwright | playwright | Pure CDP    |
 * |---------------------------|----------------------|------------|-------------|
 * | page.evaluate() context   | ISOLATED             | MAIN       | Configurable|
 * | exposeBinding() context   | ISOLATED             | MAIN       | N/A         |
 * | History API wrapping      | Only via HTML inject | Works      | Full control|
 * | Anti-detection patches    | Yes                  | No         | Manual      |
 * | CDP access                | Full                 | Full       | Native      |
 * | Service worker support    | Full                 | Full       | Full        |
 * | Multi-browser support     | Chromium only        | All        | Chromium    |
 *
 * ## Recording Architecture by Provider
 *
 * ### rebrowser-playwright (current)
 * ```
 * [Script injected via HTML] --> [fetch() to /__vrooli_recording_event__]
 *                                            |
 *                                            v
 *                            [Route intercept] --> [Event handler]
 * ```
 *
 * ### standard playwright (alternative)
 * ```
 * [Script via addInitScript] --> [window.__recordAction()] --> [exposeBinding]
 *                                                                    |
 *                                                                    v
 *                                                            [Event handler]
 * ```
 *
 * @example
 * ```typescript
 * // Get the configured provider
 * import { playwrightProvider } from './playwright';
 *
 * // Launch browser
 * const browser = await playwrightProvider.chromium.launch();
 *
 * // Check capabilities for conditional logic
 * if (playwrightProvider.capabilities.evaluateIsolated) {
 *   // Use HTML injection for script
 * } else {
 *   // Use addInitScript directly
 * }
 * ```
 */
export interface PlaywrightProvider {
  /**
   * Provider identifier for logging and debugging.
   */
  name: PlaywrightProviderName;

  /**
   * The chromium launcher from the provider package.
   * Use this instead of importing chromium directly.
   */
  chromium: typeof ChromiumType;

  /**
   * Capability flags documenting provider behavior.
   * Use these to adapt code to provider differences.
   */
  capabilities: PlaywrightCapabilities;
}

/**
 * Common issues when recording doesn't work and how to diagnose them.
 *
 * Use this as a reference when debugging recording problems.
 */
export const RECORDING_TROUBLESHOOTING = {
  /**
   * Events are not being captured at all.
   *
   * Likely causes:
   * 1. Script not injected (check route interception logs)
   * 2. Script crashed during initialization (check browser console)
   * 3. Event handler not set (check setEventHandler was called)
   *
   * Diagnostics:
   * - Enable diagnosticsEnabled: true in RecordingContextInitializer
   * - Check injection stats via getInjectionStats()
   * - Use verifyScriptInjection() to check script state
   */
  noEvents: {
    symptom: 'No events captured during recording',
    checkList: [
      'Verify script injection via verifyScriptInjection(page)',
      'Check context-initializer logs for injection failures',
      'Verify setEventHandler was called before user interaction',
      'Check browser console for JavaScript errors',
    ],
  },

  /**
   * History/navigation events not captured.
   *
   * This is the classic context isolation issue.
   *
   * Likely causes:
   * 1. Script running in ISOLATED context instead of MAIN
   * 2. History API wrapper not applied
   *
   * Diagnostics:
   * - Check inMainContext in verifyScriptInjection()
   * - If false, the HTML injection failed
   */
  noNavigationEvents: {
    symptom: 'Click/type events work but pushState/replaceState not captured',
    checkList: [
      'Verify inMainContext is true via verifyScriptInjection()',
      'Check that route interception is injecting into HTML',
      'Verify script tag appears in page source (View Source in browser)',
    ],
  },

  /**
   * Duplicate events captured.
   *
   * Likely causes:
   * 1. Script injected multiple times without cleanup
   * 2. Multiple event handlers registered
   *
   * Diagnostics:
   * - Check handlersCount in verifyScriptInjection()
   * - Should match expected count (typically 7-10 handlers)
   */
  duplicateEvents: {
    symptom: 'Each user action produces multiple identical events',
    checkList: [
      'Check handlersCount is reasonable (7-12)',
      'Verify cleanup function called on re-injection',
      'Check for multiple setEventHandler calls',
    ],
  },

  /**
   * Events delayed or batched unexpectedly.
   *
   * Likely causes:
   * 1. Debouncing configuration too aggressive
   * 2. Network issues with route interception
   *
   * Diagnostics:
   * - Check debounce settings in selector-config.ts
   * - Monitor fetch timing to /__vrooli_recording_event__
   */
  delayedEvents: {
    symptom: 'Events arrive late or in batches',
    checkList: [
      'Verify INPUT_DEBOUNCE_MS, SCROLL_DEBOUNCE_MS settings',
      'Check for slow route handling in event route',
    ],
  },
} as const;
