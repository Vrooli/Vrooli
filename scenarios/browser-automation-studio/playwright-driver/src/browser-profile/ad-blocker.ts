/**
 * Ad Blocker Module
 *
 * Provides configurable ad blocking using @ghostery/adblocker-playwright.
 * Supports two modes:
 * - 'ads_only': Blocks advertisements only (EasyList + uBlock filters)
 * - 'ads_and_tracking': Blocks ads + tracking scripts (includes privacy lists)
 *
 * Uses context-level route interception to support domain whitelisting.
 * This approach allows blocking to be dynamically enabled/disabled based on
 * the current page URL, even during same-tab navigation.
 */

import { PlaywrightBlocker, Request as AdblockerRequest } from '@ghostery/adblocker-playwright';
import type { BrowserContext, Route, Request } from 'rebrowser-playwright';
import type { AdBlockingMode } from '../types/browser-profile';
import { logger } from '../utils';

// Cache for blocker instances by mode (avoids reloading filter lists)
const blockerCache: Map<string, PlaywrightBlocker> = new Map();

// Track in-flight blocker creation to prevent duplicate initialization
const blockerPromises: Map<string, Promise<PlaywrightBlocker>> = new Map();

/**
 * Get or create a blocker instance for the specified mode.
 * Uses a cache to avoid reloading filter lists on every context.
 *
 * @param mode - The blocking mode ('ads_only' or 'ads_and_tracking')
 * @returns A configured PlaywrightBlocker instance
 */
async function getBlocker(mode: 'ads_only' | 'ads_and_tracking'): Promise<PlaywrightBlocker> {
  // Return cached instance if available
  const cached = blockerCache.get(mode);
  if (cached) {
    return cached;
  }

  // Check if creation is already in progress (prevent duplicate fetches)
  const pending = blockerPromises.get(mode);
  if (pending) {
    return pending;
  }

  // Create new blocker instance
  const promise = (async () => {
    logger.debug('Initializing ad blocker', { mode });
    const startTime = Date.now();

    const blocker =
      mode === 'ads_only'
        ? await PlaywrightBlocker.fromPrebuiltAdsOnly(fetch)
        : await PlaywrightBlocker.fromPrebuiltAdsAndTracking(fetch);

    const elapsed = Date.now() - startTime;
    logger.info('Ad blocker initialized', { mode, elapsedMs: elapsed });

    blockerCache.set(mode, blocker);
    return blocker;
  })();

  blockerPromises.set(mode, promise);

  try {
    return await promise;
  } finally {
    blockerPromises.delete(mode);
  }
}

/**
 * Check if a URL's hostname matches any pattern in the whitelist.
 * Supports exact matches and wildcard patterns (e.g., "*.google.com").
 *
 * @param url - The URL to check
 * @param whitelist - Array of domain patterns to match against
 * @returns True if the URL matches any whitelist pattern
 */
function isUrlWhitelisted(url: string, whitelist: string[]): boolean {
  if (!whitelist.length) return false;

  try {
    const hostname = new URL(url).hostname;
    return whitelist.some((pattern) => {
      // Handle wildcard patterns like "*.google.com"
      if (pattern.startsWith('*.')) {
        const suffix = pattern.slice(1); // ".google.com"
        const baseDomain = pattern.slice(2); // "google.com"
        return hostname.endsWith(suffix) || hostname === baseDomain;
      }
      // Exact match or subdomain match
      return hostname === pattern || hostname.endsWith('.' + pattern);
    });
  } catch {
    // Invalid URL - don't whitelist
    return false;
  }
}

/**
 * Map Playwright resource types to adblocker request types.
 * The adblocker library uses specific type strings for matching.
 */
function mapResourceType(
  type: string
): 'document' | 'stylesheet' | 'image' | 'media' | 'font' | 'script' | 'xhr' | 'fetch' | 'websocket' | 'other' {
  const mapping: Record<
    string,
    'document' | 'stylesheet' | 'image' | 'media' | 'font' | 'script' | 'xhr' | 'fetch' | 'websocket' | 'other'
  > = {
    document: 'document',
    stylesheet: 'stylesheet',
    image: 'image',
    media: 'media',
    font: 'font',
    script: 'script',
    xhr: 'xhr',
    fetch: 'fetch',
    websocket: 'websocket',
  };
  return mapping[type] || 'other';
}

/**
 * Apply ad blocking to a browser context using route interception.
 *
 * This implementation uses context-level route interception instead of
 * page-level script injection. This approach supports domain whitelisting
 * that works correctly during same-tab navigation (the page URL is checked
 * on every request, not just when the page is first created).
 *
 * @param context - The browser context to apply blocking to
 * @param mode - The blocking mode ('ads_only' or 'ads_and_tracking')
 * @param whitelist - Optional array of domain patterns to exclude from blocking
 */
export async function applyAdBlocking(
  context: BrowserContext,
  mode: 'ads_only' | 'ads_and_tracking',
  whitelist?: string[]
): Promise<void> {
  const blocker = await getBlocker(mode);
  const domainWhitelist = whitelist ?? [];

  // Use context-level route interception for whitelist support
  await context.route('**/*', async (route: Route, request: Request) => {
    try {
      // Get the page's current URL to check whitelist
      const frame = request.frame();
      const pageUrl = frame?.url() ?? '';

      // Allow all requests if page is on a whitelisted domain
      if (domainWhitelist.length > 0 && pageUrl && isUrlWhitelisted(pageUrl, domainWhitelist)) {
        return route.continue();
      }

      // Get request details for matching
      const requestUrl = request.url();
      const resourceType = mapResourceType(request.resourceType());

      // Create adblocker request for matching
      const adblockerRequest = AdblockerRequest.fromRawDetails({
        url: requestUrl,
        type: resourceType,
        sourceUrl: pageUrl || undefined,
      });

      // Check if request should be blocked
      const matchResult = blocker.match(adblockerRequest);

      if (matchResult.match) {
        logger.debug('Ad blocker: blocking request', {
          url: requestUrl,
          type: resourceType,
          filter: matchResult.filter?.rawLine,
        });
        return route.abort();
      }

      return route.continue();
    } catch (error) {
      // On error, allow the request to proceed
      logger.warn('Ad blocker: error processing request', {
        url: request.url(),
        error: error instanceof Error ? error.message : String(error),
      });
      return route.continue();
    }
  });

  logger.debug('Ad blocking applied to context (route-based)', {
    mode,
    whitelistDomains: domainWhitelist.length,
    whitelist: domainWhitelist,
  });
}

/**
 * Check if ad blocking is enabled for a given mode.
 *
 * @param mode - The ad blocking mode to check
 * @returns True if the mode enables ad blocking
 */
export function isAdBlockingEnabled(mode: AdBlockingMode | undefined): boolean {
  return mode !== undefined && mode !== 'none';
}
