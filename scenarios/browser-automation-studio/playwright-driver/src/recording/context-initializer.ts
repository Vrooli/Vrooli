/**
 * Recording Context Initializer
 *
 * Handles one-time context-level setup for recording:
 * 1. Adds init script via context.addInitScript() - runs in MAIN context
 * 2. Sets up context.exposeBinding() for event communication
 *
 * ARCHITECTURE:
 * - This runs ONCE per context (not per page, not per recording session)
 * - Individual recording sessions use message passing to activate/deactivate
 * - The init script is always present but dormant until activated
 * - Works correctly with rebrowser-playwright's isolated context patches
 *
 * WHY THIS DESIGN:
 * - rebrowser-playwright runs page.evaluate() in isolated contexts to avoid bot detection
 * - Isolated contexts can't properly wrap History API (pushState, replaceState)
 * - context.addInitScript() runs in MAIN context, so History wrapping works
 * - Message-based activation allows dynamic start/stop without re-injection
 *
 * @see init-script-generator.ts - Generates the init script
 * @see controller.ts - Uses this for recording sessions
 */

import type { BrowserContext, Page } from 'rebrowser-playwright';
import type winston from 'winston';
import {
  generateRecordingInitScript,
  DEFAULT_RECORDING_BINDING_NAME,
} from './init-script-generator';
import type { RawBrowserEvent } from './types';
import { logger as defaultLogger, LogContext, scopedLog } from '../utils';
import { waitForScriptReady } from './verification';
import { playwrightProvider } from '../playwright';
import { TEST_PAGE_HTML, TEST_PAGE_URL } from './self-test';

// =============================================================================
// Types
// =============================================================================

/**
 * Handler function for recording events received from the browser.
 */
export type RecordingEventHandler = (event: RawBrowserEvent) => void;

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
  /** Breakdown of injection methods used */
  methods: {
    head: number;
    HEAD: number;
    doctype: number;
    prepend: number;
  };
}

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
 * Options for initializing the recording context.
 */
export interface RecordingContextOptions {
  /** Custom binding name (defaults to DEFAULT_RECORDING_BINDING_NAME) */
  bindingName?: string;
  /** Logger instance */
  logger?: winston.Logger;
  /** Enable verbose diagnostics (more logging) */
  diagnosticsEnabled?: boolean;
  /**
   * Run a sanity check on the first page load to verify recording is working.
   *
   * When enabled, the initializer will:
   * 1. Wait for the first successful HTML injection
   * 2. Run verification to check script injection and context
   * 3. Log the result (warning if issues found)
   *
   * This is useful during development to catch issues early without
   * requiring human-in-the-loop testing.
   *
   * Default: false
   */
  runSanityCheck?: boolean;
  /**
   * Callback when sanity check completes.
   * Only called if runSanityCheck is true.
   */
  onSanityCheckComplete?: (result: SanityCheckResult) => void;
}

/**
 * Result of the sanity check run on first page load.
 */
export interface SanityCheckResult {
  /** Whether recording appears ready */
  ready: boolean;
  /** Timestamp of the check */
  timestamp: string;
  /** URL that was checked */
  url: string;
  /** How long the check took (ms) */
  durationMs: number;
  /** Script verification details */
  scriptVerification?: {
    loaded: boolean;
    ready: boolean;
    inMainContext: boolean;
    handlersCount: number;
    version: string | null;
    error?: string;
  };
  /** Issues found during the check */
  issues: string[];
  /** Provider being used */
  provider: string;
}

// =============================================================================
// RecordingContextInitializer
// =============================================================================

/**
 * Initializes recording capability on a browser context.
 *
 * Usage:
 * ```typescript
 * // During context creation
 * const initializer = new RecordingContextInitializer();
 * await initializer.initialize(context);
 *
 * // When recording starts
 * initializer.setEventHandler((event) => {
 *   console.log('Recorded:', event);
 * });
 *
 * // When recording stops
 * initializer.clearEventHandler();
 * ```
 */
export class RecordingContextInitializer {
  private initialized = false;
  private eventHandler: RecordingEventHandler | null = null;
  private readonly bindingName: string;
  private readonly logger: winston.Logger;
  private readonly diagnosticsEnabled: boolean;
  private readonly runSanityCheck: boolean;
  private readonly onSanityCheckComplete?: (result: SanityCheckResult) => void;
  private sanityCheckRun = false;
  private context: BrowserContext | null = null;

  /**
   * Statistics tracking for injection attempts.
   * Useful for debugging injection issues.
   */
  private injectionStats: InjectionStats = {
    attempted: 0,
    successful: 0,
    failed: 0,
    skipped: 0,
    methods: {
      head: 0,
      HEAD: 0,
      doctype: 0,
      prepend: 0,
    },
  };

  /**
   * Statistics tracking for route handler event processing.
   * Useful for diagnosing event flow issues.
   */
  private routeHandlerStats: RouteHandlerStats = {
    eventsReceived: 0,
    eventsProcessed: 0,
    eventsDroppedNoHandler: 0,
    eventsWithErrors: 0,
    lastEventAt: null,
    lastEventType: null,
  };

  /** Track pages that have event routes set up */
  private pagesWithEventRoute: WeakSet<Page> = new WeakSet();

  /** Track pages that have navigation listeners set up */
  private pagesWithNavigationListener: WeakSet<Page> = new WeakSet();

  constructor(options: RecordingContextOptions = {}) {
    this.bindingName = options.bindingName ?? DEFAULT_RECORDING_BINDING_NAME;
    this.logger = options.logger ?? defaultLogger;
    this.diagnosticsEnabled = options.diagnosticsEnabled ?? false;
    this.runSanityCheck = options.runSanityCheck ?? false;
    this.onSanityCheckComplete = options.onSanityCheckComplete;
  }

  /**
   * Get current injection statistics.
   * Returns a copy to prevent external mutation.
   */
  getInjectionStats(): InjectionStats {
    return {
      ...this.injectionStats,
      methods: { ...this.injectionStats.methods },
    };
  }

  /**
   * Get current route handler statistics.
   * Returns a copy to prevent external mutation.
   */
  getRouteHandlerStats(): RouteHandlerStats {
    return { ...this.routeHandlerStats };
  }

  /**
   * Check if an event handler is currently set.
   * Useful for diagnostics to verify the event pipeline is connected.
   */
  hasEventHandler(): boolean {
    return this.eventHandler !== null;
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
   *
   * NOTE: With rebrowser-playwright, page routes may not persist across navigation.
   * Use force=true to re-register the route after navigation.
   *
   * @param page - The page to setup event interception on
   * @param options - Options for route setup
   * @param options.force - Force re-registration even if already set up
   */
  async setupPageEventRoute(page: Page, options: { force?: boolean } = {}): Promise<void> {
    const { force = false } = options;

    // Skip if already set up for this page (unless force is true)
    if (!force && this.pagesWithEventRoute.has(page)) {
      this.logger.debug(scopedLog(LogContext.RECORDING, 'page event route already set up, skipping'), {
        url: page.url()?.slice(0, 50),
      });
      return;
    }

    const RECORDING_EVENT_URL = '/__vrooli_recording_event__';

    await page.route(`**${RECORDING_EVENT_URL}`, async (route) => {
      try {
        const request = route.request();

        // Track that we received an event
        this.routeHandlerStats.eventsReceived++;
        this.routeHandlerStats.lastEventAt = new Date().toISOString();

        this.logger.info(scopedLog(LogContext.RECORDING, 'page-level event route matched'), {
          url: request.url().slice(0, 100),
          method: request.method(),
          eventsReceived: this.routeHandlerStats.eventsReceived,
        });

        const postData = request.postData();

        if (postData) {
          const rawEvent = JSON.parse(postData) as RawBrowserEvent;
          this.routeHandlerStats.lastEventType = rawEvent.actionType;

          this.logger.info(scopedLog(LogContext.RECORDING, 'event received via page route'), {
            actionType: rawEvent.actionType,
            url: rawEvent.url?.slice(0, 50),
            hasHandler: !!this.eventHandler,
          });

          if (this.eventHandler) {
            try {
              this.eventHandler(rawEvent);
              this.routeHandlerStats.eventsProcessed++;
            } catch (error) {
              this.routeHandlerStats.eventsWithErrors++;
              this.logger.error(scopedLog(LogContext.RECORDING, 'event handler error'), {
                error: error instanceof Error ? error.message : String(error),
                actionType: rawEvent.actionType,
              });
            }
          } else {
            this.routeHandlerStats.eventsDroppedNoHandler++;
            this.logger.warn(scopedLog(LogContext.RECORDING, 'event dropped - no handler set'), {
              actionType: rawEvent.actionType,
              eventsDropped: this.routeHandlerStats.eventsDroppedNoHandler,
            });
          }
        }

        // Respond immediately to not block the page
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: '{"ok":true}',
        });
      } catch (error) {
        this.logger.error(scopedLog(LogContext.RECORDING, 'page event route handler error'), {
          error: error instanceof Error ? error.message : String(error),
        });
        await route.fulfill({ status: 500, body: 'error' });
      }
    });

    // Mark this page as having the event route set up
    this.pagesWithEventRoute.add(page);

    this.logger.info(scopedLog(LogContext.RECORDING, 'page-level event route set up'), {
      url: page.url()?.slice(0, 50),
    });
  }

  /**
   * Set up navigation listener on a page to re-register event routes after navigation.
   *
   * CRITICAL: With rebrowser-playwright, page.route() handlers do NOT persist across navigation.
   * This means after every page.goto() or link click that navigates to a new URL, the
   * event route handler is lost. This listener re-registers the route after each navigation.
   *
   * This is idempotent - safe to call multiple times for the same page.
   *
   * @param page - The page to set up the navigation listener on
   */
  setupPageNavigationListener(page: Page): void {
    // Skip if already set up for this page
    if (this.pagesWithNavigationListener.has(page)) {
      return;
    }

    // Use 'load' event to re-register routes after navigation
    // The 'load' event fires when the page has finished loading, which is the right time
    // to set up routes for event interception
    page.on('load', async () => {
      this.logger.debug(scopedLog(LogContext.RECORDING, 'page load detected, re-registering event route'), {
        url: page.url()?.slice(0, 50),
      });

      try {
        // Force re-registration since the route was lost during navigation
        await this.setupPageEventRoute(page, { force: true });
      } catch (err) {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to re-register event route after navigation'), {
          url: page.url()?.slice(0, 50),
          error: err instanceof Error ? err.message : String(err),
        });
      }
    });

    // Mark this page as having the navigation listener set up
    this.pagesWithNavigationListener.add(page);

    this.logger.debug(scopedLog(LogContext.RECORDING, 'page navigation listener set up'), {
      url: page.url()?.slice(0, 50),
    });
  }

  /**
   * Initialize recording capability on a browser context.
   *
   * This sets up:
   * 1. The recording init script (runs in MAIN context on all page loads)
   * 2. The context binding for receiving events from pages
   *
   * Safe to call multiple times (idempotent).
   *
   * @param context - The browser context to initialize
   */
  async initialize(context: BrowserContext): Promise<void> {
    if (this.initialized) {
      this.logger.debug(scopedLog(LogContext.RECORDING, 'context already initialized, skipping'));
      return;
    }

    this.logger.debug(scopedLog(LogContext.RECORDING, 'initializing recording context'), {
      bindingName: this.bindingName,
      runSanityCheck: this.runSanityCheck,
    });

    // Store context for sanity check
    this.context = context;

    // Event URL constant for recording events
    const RECORDING_EVENT_URL = '/__vrooli_recording_event__';

    this.logger.info(scopedLog(LogContext.RECORDING, 'route interception set up'), {
      eventUrl: RECORDING_EVENT_URL,
    });

    // Generate the recording script once
    const initScript = generateRecordingInitScript(this.bindingName);

    // Set up test page route - serves a special test page for automated pipeline testing
    // This allows the self-test to navigate to a page we control and verify the full pipeline
    await context.route(`**${TEST_PAGE_URL}`, async (route) => {
      this.logger.info(scopedLog(LogContext.RECORDING, 'serving test page for pipeline testing'));

      // Inject the recording script into the test page HTML
      const scriptTag = `<script>${initScript}</script>`;
      const testPageWithScript = TEST_PAGE_HTML.replace('<head>', `<head>${scriptTag}`);

      await route.fulfill({
        status: 200,
        contentType: 'text/html',
        body: testPageWithScript,
      });

      // Track as successful injection
      this.injectionStats.attempted++;
      this.injectionStats.successful++;
      this.injectionStats.methods.head++;
    });

    // Inject recording script into HTML responses via route interception
    // This ensures the script runs in MAIN context (not isolated) by being
    // part of the actual page HTML, bypassing rebrowser-playwright's context isolation
    //
    // NOTE: This route ONLY handles document (HTML) injection. Event interception
    // is handled via page.route() because rebrowser-playwright doesn't intercept
    // fetch/XHR requests via context.route() - only navigation requests work.
    // See setupPageEventRoute() for event handling.
    await context.route('**/*', async (route) => {
      const request = route.request();
      const url = request.url();

      // Handle test page URLs via fallback - navigation requests work correctly with fallback
      if (url.includes('__vrooli_recording_test__')) {
        this.logger.debug(scopedLog(LogContext.RECORDING, 'passing test page URL to dedicated route'), {
          url: url.slice(0, 100),
        });
        await route.fallback();
        return;
      }

      // Only intercept document requests (HTML pages)
      if (request.resourceType() !== 'document') {
        this.injectionStats.skipped++;
        await route.continue();
        return;
      }

      // Track injection attempt
      this.injectionStats.attempted++;

      if (this.diagnosticsEnabled) {
        this.logger.debug(scopedLog(LogContext.INJECTION, 'intercepting document request'), {
          url: url.slice(0, 80),
          resourceType: request.resourceType(),
          attemptNumber: this.injectionStats.attempted,
        });
      }

      try {
        // IMPORTANT: Don't follow redirects! If we follow redirects, the browser's URL
        // won't match the final content URL. JavaScript checking location.href would see
        // the original URL, not the redirect destination, which can cause redirect loops.
        // Let the browser handle redirects naturally - we'll inject on the final destination.
        const response = await route.fetch({
          maxRedirects: 0,
          timeout: 30000, // 30s timeout to prevent hanging
        });

        // If it's a redirect, pass it through without modification
        const status = response.status();
        if (status >= 300 && status < 400) {
          this.injectionStats.skipped++;
          if (this.diagnosticsEnabled) {
            this.logger.debug(scopedLog(LogContext.INJECTION, 'passing redirect through'), {
              url: url.slice(0, 80),
              status,
              location: response.headers()['location']?.slice(0, 80),
            });
          }
          await route.fulfill({ response });
          return;
        }

        const contentType = response.headers()['content-type'] || '';

        // Only inject into HTML responses
        if (!contentType.includes('text/html')) {
          this.injectionStats.skipped++;
          await route.fulfill({ response });
          return;
        }

        let body = await response.text();

        // Inject the recording script at the start of <head> or after <!DOCTYPE>
        const scriptTag = `<script>${initScript}</script>`;
        let injectionMethod: 'head' | 'HEAD' | 'doctype' | 'prepend';

        if (body.includes('<head>')) {
          body = body.replace('<head>', `<head>${scriptTag}`);
          injectionMethod = 'head';
        } else if (body.includes('<HEAD>')) {
          body = body.replace('<HEAD>', `<HEAD>${scriptTag}`);
          injectionMethod = 'HEAD';
        } else if (body.toLowerCase().includes('<!doctype')) {
          // Insert after doctype
          body = body.replace(/<!doctype[^>]*>/i, (match) => `${match}${scriptTag}`);
          injectionMethod = 'doctype';
        } else {
          // Prepend to body
          body = scriptTag + body;
          injectionMethod = 'prepend';
        }

        // Update stats
        this.injectionStats.successful++;
        this.injectionStats.methods[injectionMethod]++;

        // Log injection success
        if (this.diagnosticsEnabled) {
          this.logger.debug(scopedLog(LogContext.INJECTION, 'script injected successfully'), {
            url: request.url().slice(0, 80),
            method: injectionMethod,
            contentLength: body.length,
            stats: this.getInjectionStats(),
          });
        } else {
          this.logger.info(scopedLog(LogContext.INJECTION, 'script injected'), {
            url: request.url().slice(0, 80),
            method: injectionMethod,
          });
        }

        await route.fulfill({
          response,
          body,
        });

        // Trigger sanity check on first successful injection
        if (this.runSanityCheck && !this.sanityCheckRun) {
          // Use setImmediate to not block the route handler
          setImmediate(() => {
            this.triggerSanityCheck().catch((err) => {
              this.logger.error(scopedLog(LogContext.RECORDING, 'sanity check failed'), {
                error: err instanceof Error ? err.message : String(err),
              });
            });
          });
        }
      } catch (error) {
        // Track failure
        this.injectionStats.failed++;

        // Log injection failure
        this.logger.error(scopedLog(LogContext.INJECTION, 'injection failed'), {
          url: request.url().slice(0, 80),
          error: error instanceof Error ? error.message : String(error),
          stats: this.getInjectionStats(),
        });

        // Continue with original request
        await route.continue();
      }
    });

    this.logger.info(scopedLog(LogContext.RECORDING, 'HTML injection route set up'));

    // CRITICAL: Setup page-level event routes and navigation listeners for all existing pages
    // rebrowser-playwright doesn't intercept fetch/XHR via context.route(),
    // so we MUST use page.route() to catch recording events from within pages.
    // Additionally, page.route() handlers don't persist across navigation with rebrowser-playwright,
    // so we need navigation listeners to re-register routes after each navigation.
    const existingPages = context.pages();
    for (const page of existingPages) {
      try {
        await this.setupPageEventRoute(page);
        this.setupPageNavigationListener(page);
      } catch (err) {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to setup event route for existing page'), {
          url: page.url()?.slice(0, 50),
          error: err instanceof Error ? err.message : String(err),
        });
      }
    }

    // Setup event route and navigation listener for any new pages created in this context
    context.on('page', async (page: Page) => {
      this.logger.debug(scopedLog(LogContext.RECORDING, 'new page created, setting up event route and navigation listener'), {
        url: page.url()?.slice(0, 50),
      });
      try {
        await this.setupPageEventRoute(page);
        this.setupPageNavigationListener(page);
      } catch (err) {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to setup event route for new page'), {
          url: page.url()?.slice(0, 50),
          error: err instanceof Error ? err.message : String(err),
        });
      }
    });

    this.initialized = true;
    this.logger.info(scopedLog(LogContext.RECORDING, 'recording context initialized'), {
      bindingName: this.bindingName,
    });
  }

  /**
   * Set the handler for recording events.
   *
   * Called by RecordModeController when recording starts.
   * Only one handler can be active at a time.
   *
   * @param handler - Function to receive recording events
   */
  setEventHandler(handler: RecordingEventHandler): void {
    this.eventHandler = handler;
    this.logger.debug(scopedLog(LogContext.RECORDING, 'event handler set'));
  }

  /**
   * Clear the event handler.
   *
   * Called when recording stops. Events will be silently dropped
   * until a new handler is set.
   */
  clearEventHandler(): void {
    this.eventHandler = null;
    this.logger.debug(scopedLog(LogContext.RECORDING, 'event handler cleared'));
  }

  /**
   * Check if this context has been initialized.
   */
  isInitialized(): boolean {
    return this.initialized;
  }

  /**
   * Get the binding name used for this initializer.
   */
  getBindingName(): string {
    return this.bindingName;
  }

  /**
   * Run a sanity check on a page to verify recording is working.
   *
   * This can be called manually or is run automatically on first page load
   * if runSanityCheck option is enabled.
   *
   * The sanity check verifies:
   * 1. Script was injected (loaded marker set)
   * 2. Script initialized successfully (ready marker set)
   * 3. Script is running in MAIN context (required for History API)
   * 4. Handlers are registered (events will be captured)
   *
   * @param page - The page to check
   * @returns Sanity check result
   */
  async runSanityCheckOnPage(page: Page): Promise<SanityCheckResult> {
    const startTime = Date.now();
    const url = page.url();
    const issues: string[] = [];

    this.logger.debug(scopedLog(LogContext.RECORDING, 'running sanity check'), { url });

    try {
      // Wait for script to be ready (polls with timeout)
      const verification = await waitForScriptReady(page, 5000);

      const result: SanityCheckResult = {
        ready: false,
        timestamp: new Date().toISOString(),
        url,
        durationMs: Date.now() - startTime,
        scriptVerification: {
          loaded: verification.loaded,
          ready: verification.ready,
          inMainContext: verification.inMainContext,
          handlersCount: verification.handlersCount,
          version: verification.version,
          error: verification.error,
        },
        issues,
        provider: playwrightProvider.name,
      };

      // Check for issues
      if (!verification.loaded) {
        issues.push(
          `Script not loaded. This likely means HTML injection failed. ` +
            `Check route interception and that the page was navigated via HTTP(S). ` +
            `Error: ${verification.error || 'unknown'}`
        );
      } else if (verification.initError) {
        issues.push(
          `Script crashed during initialization: ${verification.initError}. ` +
            `Check browser console for details.`
        );
      } else if (!verification.ready) {
        issues.push(
          `Script loaded but not ready. Handlers registered: ${verification.handlersCount}. ` +
            `Script may have partially initialized before crashing.`
        );
      } else if (!verification.inMainContext) {
        issues.push(
          `Script running in ISOLATED context instead of MAIN. ` +
            `This means History API navigation events will NOT be captured. ` +
            `The script should be injected via HTML, not page.evaluate().`
        );
      } else if (verification.handlersCount < 7) {
        issues.push(
          `Low handler count (${verification.handlersCount}). ` +
            `Expected 7+ handlers. Some event types may not be captured.`
        );
      }

      // Determine overall ready status
      result.ready =
        verification.loaded &&
        verification.ready &&
        verification.inMainContext &&
        !verification.initError;

      // Log the result
      if (result.ready) {
        this.logger.info(scopedLog(LogContext.RECORDING, 'sanity check PASSED'), {
          url: url.slice(0, 80),
          durationMs: result.durationMs,
          handlersCount: verification.handlersCount,
          version: verification.version,
        });
      } else {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'sanity check FAILED'), {
          url: url.slice(0, 80),
          durationMs: result.durationMs,
          issues,
          verification: result.scriptVerification,
        });
      }

      return result;
    } catch (error) {
      const result: SanityCheckResult = {
        ready: false,
        timestamp: new Date().toISOString(),
        url,
        durationMs: Date.now() - startTime,
        issues: [`Sanity check error: ${error instanceof Error ? error.message : String(error)}`],
        provider: playwrightProvider.name,
      };

      this.logger.error(scopedLog(LogContext.RECORDING, 'sanity check ERROR'), {
        url: url.slice(0, 80),
        error: error instanceof Error ? error.message : String(error),
      });

      return result;
    }
  }

  /**
   * Internal method to trigger sanity check after first injection.
   * Called from the route handler when runSanityCheck is enabled.
   */
  private async triggerSanityCheck(): Promise<void> {
    if (!this.runSanityCheck || this.sanityCheckRun || !this.context) {
      return;
    }

    this.sanityCheckRun = true;

    // Get the first page from context
    const pages = this.context.pages();
    if (pages.length === 0) {
      this.logger.debug(scopedLog(LogContext.RECORDING, 'no pages for sanity check'));
      return;
    }

    const page = pages[0];

    // Wait a bit for the page to finish loading and script to initialize
    await new Promise((resolve) => setTimeout(resolve, 500));

    const result = await this.runSanityCheckOnPage(page);

    // Call callback if provided
    if (this.onSanityCheckComplete) {
      try {
        this.onSanityCheckComplete(result);
      } catch (error) {
        this.logger.error(scopedLog(LogContext.RECORDING, 'sanity check callback error'), {
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }
  }
}

/**
 * Create a RecordingContextInitializer.
 *
 * @param options - Initialization options
 * @returns New initializer instance
 */
export function createRecordingContextInitializer(
  options: RecordingContextOptions = {}
): RecordingContextInitializer {
  return new RecordingContextInitializer(options);
}
