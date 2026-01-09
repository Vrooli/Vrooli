/**
 * Ad Blocker Module
 *
 * Provides configurable ad blocking using @ghostery/adblocker-playwright.
 * Supports two modes:
 * - 'ads_only': Blocks advertisements only (EasyList + uBlock filters)
 * - 'ads_and_tracking': Blocks ads + tracking scripts (includes privacy lists)
 *
 * Uses Ghostery's native enableBlockingInPage() which injects blocking scripts
 * directly into pages. This approach:
 * - Does NOT use route interception (avoids conflicts with service workers)
 * - Works like browser extension ad blockers
 * - Supports domain whitelisting at the page level
 */

import { PlaywrightBlocker } from '@ghostery/adblocker-playwright';
import type { BrowserContext, Page } from 'rebrowser-playwright';
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
 * Enable ad blocking on a single page if not whitelisted.
 *
 * @param blocker - The Ghostery blocker instance
 * @param page - The page to enable blocking on
 * @param whitelist - Domain patterns to exclude from blocking
 */
async function enableBlockingOnPage(
  blocker: PlaywrightBlocker,
  page: Page,
  whitelist: string[]
): Promise<void> {
  try {
    const url = page.url();

    // Skip blocking for whitelisted domains
    if (whitelist.length > 0 && isUrlWhitelisted(url, whitelist)) {
      logger.debug('Ad blocker: skipping whitelisted page', { url });
      return;
    }

    // Skip about:blank and other special pages
    if (!url || url === 'about:blank' || url.startsWith('chrome:') || url.startsWith('devtools:')) {
      return;
    }

    // Enable blocking using Ghostery's native method (script injection, not route interception)
    await blocker.enableBlockingInPage(page);

    logger.debug('Ad blocker: enabled on page', { url });
  } catch (error) {
    // Log but don't fail - ad blocking is best-effort
    logger.warn('Ad blocker: failed to enable on page', {
      url: page.url(),
      error: error instanceof Error ? error.message : String(error),
    });
  }
}

/**
 * Apply ad blocking to a browser context using Ghostery's native script injection.
 *
 * This implementation uses Ghostery's enableBlockingInPage() which injects
 * blocking scripts directly into pages. Unlike route interception:
 * - Does NOT conflict with service workers
 * - Does NOT intercept/delay network requests
 * - Works like how browser extension ad blockers work
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

  // Enable blocking on existing pages
  for (const page of context.pages()) {
    await enableBlockingOnPage(blocker, page, domainWhitelist);
  }

  // Enable blocking on new pages as they're created
  context.on('page', async (page: Page) => {
    // Wait for page to have a URL before checking whitelist
    // This handles the case where the page is created but not yet navigated
    await enableBlockingOnPage(blocker, page, domainWhitelist);

    // Also handle navigation to new domains
    page.on('framenavigated', async (frame) => {
      // Only handle main frame navigations
      if (frame === page.mainFrame()) {
        await enableBlockingOnPage(blocker, page, domainWhitelist);
      }
    });
  });

  logger.debug('Ad blocking applied to context (script injection)', {
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
