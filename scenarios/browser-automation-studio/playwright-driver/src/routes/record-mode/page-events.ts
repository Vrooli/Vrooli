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

import type { Page, Frame } from 'rebrowser-playwright';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { createCircuitBreaker, type CircuitBreaker } from '../../infra';
import { logger, scopedLog, LogContext } from '../../utils';
import type { DriverPageEvent } from './types';
import { captureThumbnail, emitHistoryCallback, readFaviconUrl, registerRecordingPage, unregisterRecordingPage } from './recording-pages';

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
 * @returns Immediate cleanup ownership and initial-page delivery readiness
 */
export function setupPageLifecycleListeners(
  sessionId: string,
  session: ReturnType<SessionManager['getSession']>,
  pageCallbackUrl: string,
  config: Config
): { cleanup: () => void; ready: Promise<void> } {
  const context = session.context;
  const listeners = new Map<Page, () => void>();
  let active = true;
  const reportError = (error: unknown): void => {
    logger.warn(scopedLog(LogContext.RECORDING, 'page event listener failed'), {
      sessionId, error: error instanceof Error ? error.message : String(error),
    });
  };
  const event = (pageId: string, eventType: DriverPageEvent['eventType'], url = '', title = '', faviconUrl?: string): DriverPageEvent => ({
    sessionId, driverPageId: pageId, vrooliPageId: '', eventType, url, title, faviconUrl,
    timestamp: new Date().toISOString(),
  });

  const attach = (page: Page, pageId: string, admitted = Promise.resolve(true)): (() => void) | undefined => {
    if (!active || page.isClosed() || listeners.has(page)) return;
    const owns = () => active && listeners.get(page) === detach;
    const navigate = async (frame: Frame): Promise<void> => {
      if (!owns() || frame !== page.mainFrame()) return;
      const url = page.url();
      if (!await admitted || !owns()) return;
      await page.waitForLoadState('domcontentloaded', { timeout: 5000 }).catch(() => {});
      if (!owns() || page.url() !== url) return;
      const [title, faviconUrl] = await Promise.all([page.title().catch(() => ''), readFaviconUrl(page)]);
      if (!owns() || page.url() !== url) return;
      await sendPageEvent(sessionId, pageCallbackUrl, event(pageId, 'navigated', url, title, faviconUrl));
      if (!owns()) return;
      const thumbnail = config.history.thumbnailEnabled
        ? await captureThumbnail(page, config.history.thumbnailQuality)
        : undefined;
      if (owns()) await emitHistoryCallback(config, sessionId, url, title, 'navigate', thumbnail);
    };
    const close = async (): Promise<void> => {
      if (!owns()) return;
      detach();
      unregisterRecordingPage(session, page);
      if (!await admitted || !active) return;
      await sendPageEvent(sessionId, pageCallbackUrl, event(pageId, 'closed'));
    };
    const onNavigate = (frame: Frame) => navigate(frame).catch(reportError);
    const onClose = () => close().catch(reportError);
    const detach = () => {
      page.off('framenavigated', onNavigate);
      page.off('close', onClose);
      listeners.delete(page);
    };
    listeners.set(page, detach);
    page.on('framenavigated', onNavigate);
    page.on('close', onClose);
    return detach;
  };

  const newPage = async (page: Page): Promise<void> => {
    if (!active) return;
    const pageId = registerRecordingPage(session, page);
    let settle!: (published: boolean) => void;
    const admitted = new Promise<boolean>((resolve) => { settle = resolve; });
    // Observe immediately, but publish later events only after creation.
    const detach = attach(page, pageId, admitted);
    let published = false;
    try {
      const opener = await page.opener();
      if (!active) return;
      await page.waitForLoadState('domcontentloaded', { timeout: 5000 }).catch(() => {});
      if (!active || page.isClosed()) return;
      const url = page.url();
      const [title, faviconUrl] = await Promise.all([page.title().catch(() => ''), readFaviconUrl(page)]);
      if (!active || page.isClosed()) return;
      await sendPageEvent(sessionId, pageCallbackUrl, {
        ...event(pageId, 'created', url, title, page.url() === url ? faviconUrl : undefined),
        openerDriverPageId: opener ? session.pageToIdMap.get(opener) : undefined,
      });
      published = true;
    } finally {
      if (!published) detach?.();
      settle(published);
    }
  };
  const onNewPage = (page: Page) => newPage(page).catch(reportError);
  context.on('page', onNewPage);
  for (const page of session.pages) attach(page, registerRecordingPage(session, page));

  const initialPage = session.page;
  const initialId = session.pageToIdMap.get(initialPage);
  const ready = (async () => {
    if (!initialId) return;
    const url = initialPage.url();
    const [title, faviconUrl] = await Promise.all([initialPage.title().catch(() => ''), readFaviconUrl(initialPage)]);
    if (active && listeners.has(initialPage) && initialPage.url() === url) {
      await sendPageEvent(sessionId, pageCallbackUrl, event(initialId, 'initial', url, title, faviconUrl));
    }
  })();
  void ready.catch(reportError);
  const cleanup = () => {
    active = false;
    context.off('page', onNewPage);
    for (const detach of listeners.values()) detach();
    pageEventCircuitBreaker.cleanup(sessionId);
  };
  return { cleanup, ready };
}
