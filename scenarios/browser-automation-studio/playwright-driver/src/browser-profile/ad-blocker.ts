/**
 * Ad Blocker Module
 *
 * Provides configurable ad blocking using @ghostery/adblocker-playwright.
 * Supports two modes:
 * - 'ads_only': Blocks advertisements only (EasyList + uBlock filters)
 * - 'ads_and_tracking': Blocks ads + tracking scripts (includes privacy lists)
 *
 * The blocker uses filter lists compatible with uBlock Origin and EasyList,
 * providing ~99% filter coverage without requiring browser extensions.
 */

import { PlaywrightBlocker } from '@ghostery/adblocker-playwright';
import type { BrowserContext, Page } from 'playwright';
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
 * Enable ad blocking on a specific page.
 * Handles errors gracefully (e.g., page closed during setup).
 *
 * @param blocker - The PlaywrightBlocker instance
 * @param page - The page to enable blocking on
 */
async function enableBlockingOnPage(blocker: PlaywrightBlocker, page: Page): Promise<void> {
  try {
    await blocker.enableBlockingInPage(page);
  } catch (error) {
    // Page might be closed during setup - this is expected
    if (!page.isClosed()) {
      logger.warn('Failed to enable ad blocking on page', {
        url: page.url(),
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }
}

/**
 * Apply ad blocking to a browser context.
 * Sets up blocking for all current and future pages in the context.
 *
 * This function:
 * 1. Gets or creates a cached blocker instance for the specified mode
 * 2. Applies blocking to all existing pages in the context
 * 3. Sets up a listener for new pages (tabs, popups) to automatically apply blocking
 *
 * @param context - The browser context to apply blocking to
 * @param mode - The blocking mode ('ads_only' or 'ads_and_tracking')
 */
export async function applyAdBlocking(
  context: BrowserContext,
  mode: 'ads_only' | 'ads_and_tracking'
): Promise<void> {
  const blocker = await getBlocker(mode);

  // Apply to all existing pages
  const existingPages = context.pages();
  for (const page of existingPages) {
    await enableBlockingOnPage(blocker, page);
  }

  // Listen for new pages (tabs, popups)
  context.on('page', async (page) => {
    await enableBlockingOnPage(blocker, page);
  });

  logger.debug('Ad blocking applied to context', {
    mode,
    existingPages: existingPages.length,
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
