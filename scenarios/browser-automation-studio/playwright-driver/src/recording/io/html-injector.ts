/**
 * HTML Injector - Recording Script Injection via Route Interception
 *
 * SINGLE RESPONSIBILITY: Intercept HTML document requests and inject
 * the recording script into the page's HTML before it reaches the browser.
 *
 * WHY THIS EXISTS:
 *   - rebrowser-playwright runs page.evaluate() in isolated contexts
 *   - Isolated contexts can't properly wrap History API (pushState, replaceState)
 *   - By injecting into HTML, the script runs in MAIN context (not isolated)
 *   - This allows proper History API wrapping for navigation event capture
 *
 * EXTRACTED FROM: context-initializer.ts (881 lines → ~200 lines each)
 *
 * @see context-initializer.ts - Uses this for HTML injection setup
 * @see init-script-generator.ts - Generates the script being injected
 * @see decisions.ts - shouldInjectScript() decision function
 */

import type { BrowserContext, Route, Request } from 'rebrowser-playwright';
import type winston from 'winston';
import { generateRecordingInitScript } from '../capture/init-script-generator';
import { shouldInjectScript, formatDecisionForLog } from '../orchestration/decisions';
import { LogContext, scopedLog } from '../../utils';

// =============================================================================
// Types
// =============================================================================

/**
 * Statistics about script injection attempts.
 * Useful for diagnostics and debugging injection issues.
 */
export interface InjectionStats {
  /** Number of document requests where injection was attempted */
  attempted: number;
  /** Number of successful script injections */
  successful: number;
  /** Number of failed injection attempts */
  failed: number;
  /** Number of requests skipped (non-HTML, non-document) */
  skipped: number;
  /** Total injection requests (same as attempted, for convenience) */
  total: number;
  /** Breakdown of injection methods used */
  methods: {
    head: number;
    HEAD: number;
    doctype: number;
    prepend: number;
  };
}

/**
 * Creates initial injection stats object.
 */
export function createInjectionStats(): InjectionStats {
  return {
    attempted: 0,
    successful: 0,
    failed: 0,
    skipped: 0,
    total: 0,
    methods: {
      head: 0,
      HEAD: 0,
      doctype: 0,
      prepend: 0,
    },
  };
}

/**
 * Clones injection stats to prevent external mutation.
 */
export function cloneInjectionStats(stats: InjectionStats): InjectionStats {
  return {
    ...stats,
    methods: { ...stats.methods },
  };
}

/**
 * Options for setting up HTML injection.
 */
export interface HtmlInjectorOptions {
  /** Name for the exposed binding (used in script generation) */
  bindingName: string;
  /** Logger instance */
  logger: winston.Logger;
  /** Enable verbose diagnostics (more logging) */
  diagnosticsEnabled?: boolean;
  /** Callback when first successful injection occurs */
  onFirstInjection?: () => void;
}

/**
 * Result of HTML injection into a response body.
 */
interface InjectionResult {
  /** Modified HTML body with script injected */
  modifiedBody: string;
  /** Method used for injection */
  method: 'head' | 'HEAD' | 'doctype' | 'prepend';
  /** Original body length */
  originalLength: number;
  /** Modified body length */
  modifiedLength: number;
}

// =============================================================================
// Core Logic
// =============================================================================

/**
 * Inject the recording script into an HTML body.
 * Returns the modified body and injection metadata.
 *
 * @param originalBody - Original HTML response body
 * @param initScript - Generated recording init script
 * @returns Injection result with modified body and method used
 */
export function injectScriptIntoHtml(originalBody: string, initScript: string): InjectionResult {
  const scriptTag = `<script>${initScript}</script>`;
  let method: 'head' | 'HEAD' | 'doctype' | 'prepend';
  let modifiedBody: string;

  if (originalBody.includes('<head>')) {
    modifiedBody = originalBody.replace('<head>', `<head>${scriptTag}`);
    method = 'head';
  } else if (originalBody.includes('<HEAD>')) {
    modifiedBody = originalBody.replace('<HEAD>', `<HEAD>${scriptTag}`);
    method = 'HEAD';
  } else if (originalBody.toLowerCase().includes('<!doctype')) {
    // Insert after doctype
    modifiedBody = originalBody.replace(/<!doctype[^>]*>/i, (match) => `${match}${scriptTag}`);
    method = 'doctype';
  } else {
    // Prepend to body
    modifiedBody = scriptTag + originalBody;
    method = 'prepend';
  }

  return {
    modifiedBody,
    method,
    originalLength: originalBody.length,
    modifiedLength: modifiedBody.length,
  };
}

/**
 * Handle a single route request for potential HTML injection.
 *
 * This is the core logic extracted from the route handler.
 * It decides whether to inject, fetches the response, modifies it,
 * and fulfills the route.
 *
 * @returns Whether injection was successful
 */
async function handleRouteForInjection(
  route: Route,
  request: Request,
  initScript: string,
  stats: InjectionStats,
  logger: winston.Logger,
  diagnosticsEnabled: boolean
): Promise<boolean> {
  const url = request.url();
  const resourceType = request.resourceType();

  // Phase 1: Pre-fetch decision (before making network request)
  const preFetchDecision = shouldInjectScript(resourceType, url);

  if (!preFetchDecision.shouldInject) {
    // Log decision for debugging
    if (diagnosticsEnabled || preFetchDecision.reason === 'skip_test_page') {
      logger.debug(
        scopedLog(LogContext.INJECTION, `decision: ${preFetchDecision.reason}`),
        formatDecisionForLog(preFetchDecision)
      );
    }

    if (preFetchDecision.reason === 'skip_test_page') {
      await route.fallback();
    } else {
      stats.skipped++;
      await route.continue();
    }
    return false;
  }

  // Track injection attempt
  stats.attempted++;
  stats.total++;

  if (diagnosticsEnabled) {
    logger.debug(scopedLog(LogContext.INJECTION, 'intercepting document request'), {
      url: url.slice(0, 80),
      resourceType: resourceType,
      attemptNumber: stats.attempted,
    });
  }

  try {
    // Follow redirects to get the final content. With rebrowser-playwright, if we
    // return a 3xx response via route.fulfill(), the browser follows the redirect
    // but the new request does NOT go through our route handler again (anti-detection).
    // By following redirects ourselves, we ensure we can inject into the final HTML.
    //
    // Note: The browser's URL bar will show the original URL (e.g., wikipedia.com)
    // not the redirect destination (e.g., www.wikipedia.org). JavaScript on the page
    // checking location.href will see the original URL. This is generally acceptable
    // for recording purposes.

    // Step 1: Log before fetch
    logger.info(scopedLog(LogContext.INJECTION, 'fetching document for injection'), {
      url: url.slice(0, 100),
      resourceType,
    });

    const response = await route.fetch({
      maxRedirects: 10, // Follow redirects to get final content
      timeout: 30000, // 30s timeout to prevent hanging
    });

    // Phase 2: Post-fetch decision (after we have response headers)
    const status = response.status();
    const contentType = response.headers()['content-type'] || '';

    // Step 2: Log after fetch
    logger.info(scopedLog(LogContext.INJECTION, 'document fetched'), {
      url: url.slice(0, 100),
      status,
      contentType: contentType.slice(0, 50),
    });

    const postFetchDecision = shouldInjectScript(resourceType, url, status, contentType);

    if (!postFetchDecision.shouldInject) {
      // Log decision for debugging
      logger.debug(
        scopedLog(LogContext.INJECTION, `decision: ${postFetchDecision.reason}`),
        formatDecisionForLog(postFetchDecision)
      );

      stats.skipped++;
      await route.fulfill({ response });
      return false;
    }

    const originalBody = await response.text();

    // Inject the recording script
    const injection = injectScriptIntoHtml(originalBody, initScript);

    // Step 3: Log after modification
    logger.info(scopedLog(LogContext.INJECTION, 'HTML modified for injection'), {
      url: url.slice(0, 100),
      method: injection.method,
      originalLength: injection.originalLength,
      modifiedLength: injection.modifiedLength,
      scriptInjectedLength: injection.modifiedLength - injection.originalLength,
    });

    // Fulfill the route with modified body
    await route.fulfill({
      response,
      body: injection.modifiedBody,
    });

    // Step 4: Update stats AFTER successful fulfill (critical fix!)
    stats.successful++;
    stats.methods[injection.method]++;

    // Step 5: Log after successful fulfill
    logger.info(scopedLog(LogContext.INJECTION, 'route.fulfill completed - injection successful'), {
      url: url.slice(0, 100),
      method: injection.method,
      stats: cloneInjectionStats(stats),
    });

    return true;
  } catch (error) {
    // Track failure
    stats.failed++;

    // Log injection failure
    logger.error(scopedLog(LogContext.INJECTION, 'injection failed'), {
      url: request.url().slice(0, 80),
      error: error instanceof Error ? error.message : String(error),
      stats: cloneInjectionStats(stats),
    });

    // Continue with original request
    await route.continue();
    return false;
  }
}

// =============================================================================
// Public API
// =============================================================================

/**
 * Set up HTML injection route on a browser context.
 *
 * This registers a route handler that intercepts all requests and:
 * 1. Checks if it's a document (HTML) request
 * 2. Fetches the response
 * 3. Injects the recording script into the HTML
 * 4. Returns the modified response
 *
 * @param context - Browser context to set up injection on
 * @param options - Configuration options
 * @returns Object with stats getter and cleanup info
 */
export async function setupHtmlInjectionRoute(
  context: BrowserContext,
  options: HtmlInjectorOptions
): Promise<{
  getStats: () => InjectionStats;
}> {
  const { bindingName, logger, diagnosticsEnabled = false, onFirstInjection } = options;

  // Generate the recording script once
  const initScript = generateRecordingInitScript(bindingName);

  // Track injection stats
  const stats = createInjectionStats();

  // Track if we've fired the first injection callback
  let firstInjectionFired = false;

  // Set up the route handler
  await context.route('**/*', async (route) => {
    const request = route.request();

    const success = await handleRouteForInjection(
      route,
      request,
      initScript,
      stats,
      logger,
      diagnosticsEnabled
    );

    // Fire first injection callback if this was a successful injection
    if (success && !firstInjectionFired && onFirstInjection) {
      firstInjectionFired = true;
      // Use setImmediate to not block the route handler
      setImmediate(() => {
        try {
          onFirstInjection();
        } catch (err) {
          logger.error(scopedLog(LogContext.INJECTION, 'first injection callback error'), {
            error: err instanceof Error ? err.message : String(err),
          });
        }
      });
    }
  });

  logger.info(scopedLog(LogContext.RECORDING, 'HTML injection route set up'));

  return {
    getStats: () => cloneInjectionStats(stats),
  };
}
