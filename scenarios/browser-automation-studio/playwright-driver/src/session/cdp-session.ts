/**
 * CDP Session Management
 *
 * Utilities for creating and managing Chrome DevTools Protocol sessions
 * via Playwright's CDPSession API.
 *
 * CDP sessions provide direct access to Chrome's debugging protocol,
 * enabling advanced features like screencast streaming that aren't
 * available through Playwright's standard API.
 */

import type { CDPSession, Page } from 'playwright';

/**
 * Create a CDP session for a Playwright page.
 *
 * Note: CDP sessions are per-page. If the page navigates cross-origin,
 * the session remains valid but some commands may fail.
 *
 * @param page - Playwright page to create CDP session for
 * @returns CDP session instance
 */
export async function createCDPSession(page: Page): Promise<CDPSession> {
  return await page.context().newCDPSession(page);
}

/**
 * Safely detach a CDP session.
 * Handles cases where session is already detached.
 *
 * @param session - CDP session to detach
 */
export async function detachCDPSession(session: CDPSession): Promise<void> {
  try {
    await session.detach();
  } catch {
    // Session may already be detached - this is fine
  }
}

/**
 * CDP session cache for reusing sessions across operations.
 * WeakMap ensures sessions are cleaned up when pages are garbage collected.
 */
const cdpSessionCache = new WeakMap<Page, CDPSession>();

/**
 * Get or create a cached CDP session for a page.
 *
 * CDP sessions are cached to avoid per-operation creation overhead.
 * This is useful for high-frequency operations like frame capture.
 *
 * @param page - Playwright page
 * @returns Cached or new CDP session
 */
export async function getCachedCDPSession(page: Page): Promise<CDPSession> {
  let session = cdpSessionCache.get(page);
  if (!session) {
    session = await page.context().newCDPSession(page);
    cdpSessionCache.set(page, session);
  }
  return session;
}

/**
 * Clear cached CDP session for a page.
 * Call this when you need a fresh session or before page close.
 *
 * @param page - Playwright page
 */
export function clearCachedCDPSession(page: Page): void {
  cdpSessionCache.delete(page);
}
