/**
 * Recording Context Initializer
 *
 * COORDINATOR: Orchestrates one-time context-level setup for recording by
 * composing specialized modules.
 *
 * ARCHITECTURE:
 *   - This runs ONCE per context (not per page, not per recording session)
 *   - Individual recording sessions use message passing to activate/deactivate
 *   - The init script is always present but dormant until activated
 *   - Works correctly with rebrowser-playwright's isolated context patches
 *
 * COMPOSITION:
 *   - html-injector.ts - Injects recording script into HTML documents
 *   - event-route.ts - Sets up page-level routes for event interception
 *   - This file - Coordinates the above and handles sanity checks
 *
 * WHY THIS DESIGN:
 *   - rebrowser-playwright runs page.evaluate() in isolated contexts
 *   - Isolated contexts can't properly wrap History API (pushState, replaceState)
 *   - context.addInitScript() runs in MAIN context, so History wrapping works
 *   - Message-based activation allows dynamic start/stop without re-injection
 *
 * REFACTORED: Split from 881-line monolith into focused modules
 *
 * @see html-injector.ts - HTML injection route setup
 * @see event-route.ts - Page event route setup
 * @see init-script-generator.ts - Generates the init script
 * @see pipeline-manager.ts - Uses this for recording sessions
 */

import type { BrowserContext, Page } from 'rebrowser-playwright';
import type winston from 'winston';
import { DEFAULT_RECORDING_BINDING_NAME } from '../capture/init-script-generator';
import type { RawBrowserEvent } from '../types';
import { logger as defaultLogger, LogContext, scopedLog } from '../../utils';
import { waitForScriptReady } from '../validation/verification';
import { playwrightProvider } from '../../playwright';

// Import composed modules
import {
  setupHtmlInjectionRoute,
  type InjectionStats,
  createInjectionStats,
} from './html-injector';
import {
  createEventRouteManager,
  type EventRouteManager,
  type RouteHandlerStats,
  createRouteHandlerStats,
} from './event-route';

// =============================================================================
// Types - Re-export from composed modules for backward compatibility
// =============================================================================

export type { InjectionStats } from './html-injector';
export type { RouteHandlerStats, RecordingEventHandler } from './event-route';

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
 * This is the main entry point for recording infrastructure setup.
 * It coordinates HTML injection and event routing modules.
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
  private eventHandler: ((event: RawBrowserEvent) => void) | null = null;
  private readonly bindingName: string;
  private readonly logger: winston.Logger;
  private readonly diagnosticsEnabled: boolean;
  private readonly runSanityCheck: boolean;
  private readonly onSanityCheckComplete?: (result: SanityCheckResult) => void;
  private sanityCheckRun = false;
  private context: BrowserContext | null = null;

  // Composed modules
  private eventRouteManager: EventRouteManager | null = null;

  // Stats from injection module (updated after initialization)
  private injectionStatsGetter: (() => InjectionStats) | null = null;

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
    if (this.injectionStatsGetter) {
      return this.injectionStatsGetter();
    }
    return createInjectionStats();
  }

  /**
   * Get current route handler statistics.
   * Returns a copy to prevent external mutation.
   */
  getRouteHandlerStats(): RouteHandlerStats {
    if (this.eventRouteManager) {
      return this.eventRouteManager.getStats();
    }
    return createRouteHandlerStats();
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
   * Delegates to the event route manager.
   *
   * @param page - The page to setup event interception on
   * @param options - Options for route setup
   */
  async setupPageEventRoute(page: Page, options: { force?: boolean } = {}): Promise<void> {
    if (!this.eventRouteManager) {
      throw new Error('Context not initialized. Call initialize() first.');
    }
    await this.eventRouteManager.setupPageEventRoute(page, options);
  }

  /**
   * Initialize recording capability on a browser context.
   *
   * This sets up:
   * 1. The HTML injection route (injects recording script into pages)
   * 2. The event route manager (handles events from pages)
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

    // Create event route manager
    this.eventRouteManager = createEventRouteManager({
      logger: this.logger,
      getEventHandler: () => this.eventHandler,
    });

    // Set up HTML injection route
    const injectionResult = await setupHtmlInjectionRoute(context, {
      bindingName: this.bindingName,
      logger: this.logger,
      diagnosticsEnabled: this.diagnosticsEnabled,
      onFirstInjection: () => {
        if (this.runSanityCheck && !this.sanityCheckRun) {
          this.triggerSanityCheck().catch((err) => {
            this.logger.error(scopedLog(LogContext.RECORDING, 'sanity check failed'), {
              error: err instanceof Error ? err.message : String(err),
            });
          });
        }
      },
    });

    this.injectionStatsGetter = injectionResult.getStats;

    // Set up page-level event routes for all existing pages
    // NOTE: Navigation listeners are NOT set up here - they are handled by
    // pipeline-manager.ts during recording to avoid duplicate handlers.
    const existingPages = context.pages();
    for (const page of existingPages) {
      try {
        await this.eventRouteManager.setupPageEventRoute(page);
      } catch (err) {
        this.logger.warn(scopedLog(LogContext.RECORDING, 'failed to setup event route for existing page'), {
          url: page.url()?.slice(0, 50),
          error: err instanceof Error ? err.message : String(err),
        });
      }
    }

    // Setup event route for any new pages created in this context
    // NOTE: Navigation listeners are handled by pipeline-manager.ts during recording
    context.on('page', async (page: Page) => {
      this.logger.debug(scopedLog(LogContext.RECORDING, 'new page created, setting up event route'), {
        url: page.url()?.slice(0, 50),
      });
      try {
        await this.eventRouteManager!.setupPageEventRoute(page);
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
   * Called by RecordingPipelineManager when recording starts.
   * Only one handler can be active at a time.
   *
   * @param handler - Function to receive recording events
   */
  setEventHandler(handler: (event: RawBrowserEvent) => void): void {
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

  // ===========================================================================
  // Sanity Check Logic
  // ===========================================================================

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
   * Called from the HTML injector when first successful injection occurs.
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
