/**
 * Page Events
 *
 * Handles page lifecycle events for multi-tab recording sessions.
 * Tracks new tabs, navigation events, and page closes, sending events
 * to the configured callback URL.
 *
 * Uses a separate circuit breaker from callback streaming to allow
 * independent failure handling for page events vs action events.
 */

import type { Page } from 'rebrowser-playwright';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { createCircuitBreaker, type CircuitBreaker } from '../../infra';
import { logger, scopedLog, LogContext } from '../../utils';
import type { DriverPageEvent } from './types';
import { captureThumbnail, emitHistoryCallback } from './recording-pages';

// =============================================================================
// Circuit Breaker for Page Events
// =============================================================================

/**
 * Circuit breaker for page event streaming.
 * Separate from action callback to allow independent failure handling.
 */
export const pageEventCircuitBreaker: CircuitBreaker<string> = createCircuitBreaker({
  maxFailures: 5,
  resetTimeoutMs: 30_000, // 30 seconds
  name: 'pageEvent',
});

/** Timeout for page event callbacks (5 seconds) */
const PAGE_EVENT_TIMEOUT_MS = 5_000;

// =============================================================================
// Page Event Functions
// =============================================================================

/**
 * Send a page event to the callback URL with circuit breaker protection.
 */
export async function sendPageEvent(
  sessionId: string,
  callbackUrl: string,
  event: DriverPageEvent
): Promise<void> {
  // Check if we should attempt half-open (atomically claims the attempt)
  const attemptHalfOpen = pageEventCircuitBreaker.tryEnterHalfOpen(sessionId);

  // Skip if circuit is open and we're not the half-open attempt
  if (pageEventCircuitBreaker.isOpen(sessionId) && !attemptHalfOpen) {
    logger.debug(scopedLog(LogContext.RECORDING, 'skipping page event (circuit open)'), {
      sessionId,
      eventType: event.eventType,
      driverPageId: event.driverPageId,
    });
    return;
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), PAGE_EVENT_TIMEOUT_MS);

  try {
    const response = await fetch(callbackUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(event),
      signal: controller.signal,
    });

    if (!response.ok) {
      throw new Error(`Page event callback returned ${response.status}: ${response.statusText}`);
    }

    pageEventCircuitBreaker.recordSuccess(sessionId);

    logger.debug(scopedLog(LogContext.RECORDING, 'page event sent'), {
      sessionId,
      eventType: event.eventType,
      driverPageId: event.driverPageId,
      url: event.url,
    });
  } catch (err) {
    const circuitOpened = pageEventCircuitBreaker.recordFailure(sessionId);
    const errorMessage = err instanceof Error ? err.message : String(err);

    logger.warn(scopedLog(LogContext.RECORDING, 'page event send failed'), {
      sessionId,
      callbackUrl,
      eventType: event.eventType,
      error: errorMessage,
      circuitOpen: circuitOpened,
    });
  } finally {
    clearTimeout(timeoutId);
  }
}

/**
 * Set up page lifecycle listeners for a recording session.
 * Tracks new tabs, navigation events, and page closes.
 *
 * @returns Cleanup function to call when recording stops
 */
export function setupPageLifecycleListeners(
  sessionId: string,
  session: ReturnType<SessionManager['getSession']>,
  pageCallbackUrl: string,
  config: Config
): () => void {
  const context = session.context;

  // Handler for new pages (new tabs/popups)
  const onNewPage = async (newPage: Page): Promise<void> => {
    // Generate a unique ID for this page
    const pageId = crypto.randomUUID();

    // Add to session tracking
    session.pages.push(newPage);
    session.pageIdMap.set(pageId, newPage);
    session.pageToIdMap.set(newPage, pageId);

    // Find the opener page ID if any
    const openerPage = await newPage.opener();
    const openerPageId = openerPage ? session.pageToIdMap.get(openerPage) : undefined;

    // Wait for the page to be ready enough to get URL/title
    try {
      await newPage.waitForLoadState('domcontentloaded', { timeout: 5000 }).catch(() => {});
    } catch {
      // Ignore timeout - page might be blank
    }

    const url = newPage.url();
    const title = await newPage.title().catch(() => '');

    logger.info(scopedLog(LogContext.RECORDING, 'new page detected'), {
      sessionId,
      pageId,
      openerPageId,
      url,
      title,
      totalPages: session.pages.length,
    });

    // Send page_created event
    const event: DriverPageEvent = {
      sessionId,
      driverPageId: pageId,
      vrooliPageId: '',
      eventType: 'created',
      url,
      title,
      openerDriverPageId: openerPageId,
      timestamp: new Date().toISOString(),
    };

    await sendPageEvent(sessionId, pageCallbackUrl, event);

    // Set up navigation listener for this page
    newPage.on('framenavigated', async (frame) => {
      // Only track main frame navigations
      if (frame !== newPage.mainFrame()) return;

      const navUrl = newPage.url();
      const navTitle = await newPage.title().catch(() => '');

      logger.debug(scopedLog(LogContext.RECORDING, 'page navigated'), {
        sessionId,
        pageId,
        url: navUrl,
      });

      const navEvent: DriverPageEvent = {
        sessionId,
        driverPageId: pageId,
        vrooliPageId: '',
        eventType: 'navigated',
        url: navUrl,
        title: navTitle,
        timestamp: new Date().toISOString(),
      };

      await sendPageEvent(sessionId, pageCallbackUrl, navEvent);

      // Emit history callback for session profile history tracking
      const thumbnail = config.history.thumbnailEnabled
        ? await captureThumbnail(newPage, config.history.thumbnailQuality)
        : undefined;
      emitHistoryCallback(config, sessionId, navUrl, navTitle, 'navigate', thumbnail).catch(() => {
        // Error already logged in emitHistoryCallback
      });
    });

    // Set up close listener for this page
    newPage.on('close', async () => {
      logger.info(scopedLog(LogContext.RECORDING, 'page closed'), {
        sessionId,
        pageId,
      });

      const closeEvent: DriverPageEvent = {
        sessionId,
        driverPageId: pageId,
        vrooliPageId: '',
        eventType: 'closed',
        url: '',
        title: '',
        timestamp: new Date().toISOString(),
      };

      await sendPageEvent(sessionId, pageCallbackUrl, closeEvent);

      // Remove from tracking
      session.pageIdMap.delete(pageId);
      const pageIndex = session.pages.indexOf(newPage);
      if (pageIndex !== -1) {
        session.pages.splice(pageIndex, 1);
      }
    });
  };

  // Listen for new pages
  context.on('page', onNewPage);

  // Return cleanup function
  return () => {
    context.off('page', onNewPage);
    pageEventCircuitBreaker.cleanup(sessionId);
  };
}
