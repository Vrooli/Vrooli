/**
 * Event Route - Page-level Event Interception for Recording
 *
 * SINGLE RESPONSIBILITY: Set up page-level routes to intercept recording events
 * sent from the browser-side recording script.
 *
 * WHY PAGE-LEVEL ROUTES:
 *   - rebrowser-playwright doesn't intercept fetch/XHR via context.route()
 *   - Only navigation requests are intercepted at context level
 *   - We MUST use page.route() to intercept fetch requests from within pages
 *
 * ROUTE PERSISTENCE:
 *   - With rebrowser-playwright, page.route() handlers do NOT persist across navigation
 *   - Route re-registration after navigation is handled by pipeline-manager.ts
 *     (SINGLE SOURCE OF TRUTH for navigation handling during recording)
 *
 * EXTRACTED FROM: context-initializer.ts (881 lines → ~200 lines each)
 *
 * @see context-initializer.ts - Uses this for event route setup
 * @see html-injector.ts - Handles HTML injection (sibling module)
 * @see pipeline-manager.ts - Handles navigation and route re-registration
 */

import type { Page } from 'rebrowser-playwright';
import type winston from 'winston';
import { shouldProcessEvent, formatDecisionForLog } from '../orchestration/decisions';
import { LogContext, scopedLog } from '../../utils';
import { createWeakSetGuard, createSetGuard, type WeakSetGuard, type SetGuard } from '../../infra';
import type { RawBrowserEvent } from '../types';

// =============================================================================
// Constants
// =============================================================================

/** URL path for recording events */
const RECORDING_EVENT_URL = '/__vrooli_recording_event__';

// =============================================================================
// Types
// =============================================================================

/**
 * Handler function for recording events received from the browser.
 */
export type RecordingEventHandler = (event: RawBrowserEvent) => void;

/**
 * Statistics about route handler event processing.
 * Useful for diagnosing event flow issues.
 */
export interface RouteHandlerStats {
  /** Total events received by route handler */
  eventsReceived: number;
  /** Events successfully processed (handler called) */
  eventsProcessed: number;
  /** Events dropped because no handler was set */
  eventsDroppedNoHandler: number;
  /** Events that caused handler errors */
  eventsWithErrors: number;
  /** Timestamp of last event received */
  lastEventAt: string | null;
  /** Type of last event received */
  lastEventType: string | null;
}

/**
 * Creates initial route handler stats object.
 */
export function createRouteHandlerStats(): RouteHandlerStats {
  return {
    eventsReceived: 0,
    eventsProcessed: 0,
    eventsDroppedNoHandler: 0,
    eventsWithErrors: 0,
    lastEventAt: null,
    lastEventType: null,
  };
}

/**
 * Clones route handler stats to prevent external mutation.
 */
export function cloneRouteHandlerStats(stats: RouteHandlerStats): RouteHandlerStats {
  return { ...stats };
}

/**
 * Options for setting up event routes.
 */
export interface EventRouteOptions {
  /** Logger instance */
  logger: winston.Logger;
  /** Get current event handler (allows dynamic handler changes) */
  getEventHandler: () => RecordingEventHandler | null;
}

/**
 * Manages event route registration across pages.
 */
export interface EventRouteManager {
  /** Set up event route on a page */
  setupPageEventRoute: (page: Page, options?: { force?: boolean }) => Promise<void>;
  /** Get current stats */
  getStats: () => RouteHandlerStats;
  /** Check if event route is set up for a page */
  hasEventRoute: (page: Page) => boolean;
}

// =============================================================================
// Implementation
// =============================================================================

/**
 * Create an event route manager for handling recording events.
 *
 * This factory creates a manager that handles:
 * 1. Setting up page-level routes for event interception
 * 2. Re-registering routes after navigation (rebrowser-playwright quirk)
 * 3. Tracking event processing statistics
 *
 * @param options - Configuration options
 * @returns Event route manager instance
 */
export function createEventRouteManager(options: EventRouteOptions): EventRouteManager {
  const { logger, getEventHandler } = options;

  // Track statistics
  const stats = createRouteHandlerStats();

  // Track pages that have event routes set up
  const pagesWithEventRoute: WeakSetGuard<Page> = createWeakSetGuard<Page>();

  // Track pages currently being set up to prevent duplicate concurrent registrations
  const pagesBeingSetUp: SetGuard<Page> = createSetGuard<Page>({ name: 'pages-setup-lock' });

  /**
   * Handle a recording event from the page route.
   */
  async function handleRecordingEvent(postData: string): Promise<void> {
    const rawEvent = JSON.parse(postData) as RawBrowserEvent;
    stats.lastEventType = rawEvent.actionType;

    const eventHandler = getEventHandler();

    // Use decision function to determine if we should process this event
    const decision = shouldProcessEvent(
      eventHandler ? 'capturing' : 'idle',
      !!eventHandler,
      rawEvent.actionType
    );

    logger.info(
      scopedLog(LogContext.RECORDING, `event decision: ${decision.reason}`),
      formatDecisionForLog(decision)
    );

    if (decision.shouldProcess && eventHandler) {
      try {
        eventHandler(rawEvent);
        stats.eventsProcessed++;
      } catch (error) {
        stats.eventsWithErrors++;
        logger.error(scopedLog(LogContext.RECORDING, 'event handler error'), {
          error: error instanceof Error ? error.message : String(error),
          actionType: rawEvent.actionType,
        });
      }
    } else {
      stats.eventsDroppedNoHandler++;
      logger.warn(
        scopedLog(LogContext.RECORDING, `event dropped: ${decision.reason}`),
        formatDecisionForLog(decision)
      );
    }
  }

  /**
   * Setup page-level route for event interception.
   *
   * CRITICAL: rebrowser-playwright doesn't intercept fetch/XHR requests via context.route().
   * Only navigation requests are intercepted at the context level.
   * We MUST use page.route() to intercept fetch requests from within pages.
   *
   * This method:
   * 1. Adds a page-level route for the recording event URL
   * 2. Handles incoming events and forwards to the event handler
   * 3. Is idempotent - safe to call multiple times for the same page
   * 4. Uses a lock to prevent duplicate concurrent registrations
   *
   * NOTE: With rebrowser-playwright, page routes may not persist across navigation.
   * Use force=true to re-register the route after navigation.
   *
   * @param page - The page to setup event interception on
   * @param routeOptions - Options for route setup
   */
  async function setupPageEventRoute(
    page: Page,
    routeOptions: { force?: boolean } = {}
  ): Promise<void> {
    const { force = false } = routeOptions;

    // Skip if already set up for this page (unless force is true)
    if (!force && pagesWithEventRoute.has(page)) {
      logger.debug(scopedLog(LogContext.RECORDING, 'page event route already set up, skipping'), {
        url: page.url()?.slice(0, 50),
      });
      return;
    }

    // Prevent duplicate concurrent registrations using a lock
    if (!force && pagesBeingSetUp.has(page)) {
      logger.debug(scopedLog(LogContext.RECORDING, 'page event route setup already in progress, skipping'), {
        url: page.url()?.slice(0, 50),
      });
      return;
    }

    // Mark as being set up (lock acquisition)
    pagesBeingSetUp.add(page);

    try {
      await page.route(`**${RECORDING_EVENT_URL}`, async (route) => {
        try {
          const request = route.request();

          // Track that we received an event
          stats.eventsReceived++;
          stats.lastEventAt = new Date().toISOString();

          logger.info(scopedLog(LogContext.RECORDING, 'page-level event route matched'), {
            url: request.url().slice(0, 100),
            method: request.method(),
            eventsReceived: stats.eventsReceived,
          });

          const postData = request.postData();
          if (postData) {
            await handleRecordingEvent(postData);
          }

          // Respond immediately to not block the page
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: '{"ok":true}',
          });
        } catch (error) {
          logger.error(scopedLog(LogContext.RECORDING, 'page event route handler error'), {
            error: error instanceof Error ? error.message : String(error),
          });
          await route.fulfill({ status: 500, body: 'error' });
        }
      });

      // Mark this page as having the event route set up
      pagesWithEventRoute.add(page);

      logger.info(scopedLog(LogContext.RECORDING, 'page-level event route set up'), {
        url: page.url()?.slice(0, 50),
      });
    } finally {
      // Always release the lock, whether setup succeeded or failed
      pagesBeingSetUp.delete(page);
    }
  }

  // NOTE: Navigation listener (page.on('load')) is NOT set up here.
  // Route re-registration after navigation is handled by pipeline-manager.ts
  // to avoid duplicate handlers causing race conditions. See pipeline-manager.ts
  // createNavigationHandler() for the SINGLE SOURCE OF TRUTH for navigation handling.

  return {
    setupPageEventRoute,
    getStats: () => cloneRouteHandlerStats(stats),
    hasEventRoute: (page: Page) => pagesWithEventRoute.has(page),
  };
}
