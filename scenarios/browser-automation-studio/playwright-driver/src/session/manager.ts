import { chromium, Browser } from 'playwright';
import type { SessionSpec, SessionState, SessionPhase } from '../types';
import type { Config } from '../config';
import { logger, metrics, SessionNotFoundError, ResourceLimitError, scopedLog, LogContext } from '../utils';
import { buildContext } from './context-builder';
import { v4 as uuidv4 } from 'uuid';
import { removeRecordedActions } from '../recording/buffer';

/**
 * SessionManager - Manages browser session lifecycle
 *
 * Responsibilities:
 * - Session creation and deletion
 * - Session retrieval by ID
 * - Resource limit enforcement
 * - Idle timeout tracking
 * - Browser process management
 *
 * Hardened assumptions:
 * - closeSession may be called concurrently (from idle cleanup and explicit requests)
 * - Session cleanup must be idempotent
 * - Browser may disconnect unexpectedly
 *
 * Idempotency guarantees:
 * - startSession with same execution_id returns existing session (replay-safe)
 * - closeSession can be called multiple times safely (idempotent)
 * - Session creation uses in-flight tracking to prevent duplicates under concurrent calls
 */
export class SessionManager {
  private sessions: Map<string, SessionState> = new Map();
  private browser: Browser | null = null;
  private config: Config;

  private browserVerified = false;
  private browserError: string | null = null;

  /** Track sessions currently being closed to prevent double-close */
  private closingSessionIds: Set<string> = new Set();

  /**
   * Track in-flight session creation by execution_id.
   * Prevents duplicate session creation when multiple concurrent requests
   * arrive with the same execution_id before the first completes.
   */
  private pendingSessionCreations: Map<string, Promise<{ sessionId: string; reused: boolean; createdAt: Date }>> = new Map();

  /**
   * Lock to prevent concurrent browser launches.
   * Holds a promise that resolves when browser launch completes.
   * This prevents the race condition where multiple startSession() calls
   * could each launch their own browser instance.
   */
  private browserLaunchPromise: Promise<Browser> | null = null;

  constructor(config: Config) {
    this.config = config;
  }

  /**
   * Verify that the browser can be launched.
   * Called during startup to catch Chromium issues early.
   * Returns null on success, error message on failure.
   */
  async verifyBrowserLaunch(): Promise<string | null> {
    if (this.browserVerified) {
      return this.browserError;
    }

    try {
      logger.info('browser: verifying launch capability');
      const browser = await this.getBrowser();

      // Verify we can create a context and page
      const context = await browser.newContext();
      const page = await context.newPage();

      // Verify basic navigation works
      await page.goto('about:blank');

      // Cleanup verification resources
      await page.close();
      await context.close();

      this.browserVerified = true;
      this.browserError = null;

      logger.info('browser: verification successful', {
        version: browser.version(),
      });

      return null;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.browserError = errorMessage;
      this.browserVerified = true; // Mark as verified (we checked, it failed)

      logger.error('browser: verification failed', {
        error: errorMessage,
        hint: 'Check that Chromium is installed and sandbox settings are correct',
      });

      return errorMessage;
    }
  }

  /**
   * Get browser health status for health endpoint
   */
  getBrowserStatus(): { healthy: boolean; error?: string; version?: string } {
    if (!this.browserVerified) {
      return { healthy: false, error: 'Browser not yet verified' };
    }

    if (this.browserError) {
      return { healthy: false, error: this.browserError };
    }

    if (this.browser && this.browser.isConnected()) {
      return { healthy: true, version: this.browser.version() };
    }

    return { healthy: true };
  }

  /**
   * Get or create shared browser instance.
   *
   * Temporal hardening:
   * - Uses a lock (browserLaunchPromise) to prevent concurrent browser launches
   * - Multiple concurrent calls will all await the same launch promise
   * - If browser disconnects mid-launch, subsequent calls will retry
   */
  private async getBrowser(): Promise<Browser> {
    // Fast path: browser already exists and is connected
    if (this.browser && this.browser.isConnected()) {
      return this.browser;
    }

    // If another call is already launching the browser, wait for it
    if (this.browserLaunchPromise) {
      logger.debug('browser: waiting for concurrent launch to complete');
      try {
        const browser = await this.browserLaunchPromise;
        // Double-check it's still connected after await
        if (browser.isConnected()) {
          return browser;
        }
        // Browser disconnected during wait, fall through to launch new one
      } catch (error) {
        // Launch failed, fall through to try again
        logger.debug('browser: concurrent launch failed, will retry', {
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    // Create the launch promise BEFORE starting the launch
    // This ensures concurrent calls will await this promise
    this.browserLaunchPromise = this.launchBrowserInternal();

    try {
      this.browser = await this.browserLaunchPromise;
      return this.browser;
    } finally {
      // Clear the promise after launch completes (success or failure)
      // This allows retry on next call if launch failed
      this.browserLaunchPromise = null;
    }
  }

  /**
   * Internal browser launch implementation.
   * Separated from getBrowser() to make the locking logic clearer.
   */
  private async launchBrowserInternal(): Promise<Browser> {
    logger.info('browser: launching', {
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || 'auto',
    });

    const browser = await chromium.launch({
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || undefined,
      args: this.config.browser.args,
    });

    logger.info('browser: launched', { version: browser.version() });
    return browser;
  }

  /**
   * Start a new session
   *
   * Idempotency behavior:
   * - If a session with the same execution_id already exists, returns it (for reuse/clean modes)
   * - If session creation is already in-flight for this execution_id, awaits that instead of creating duplicate
   * - Uses in-flight tracking to prevent race conditions under concurrent requests
   *
   * @returns Object with session ID and whether it was reused
   */
  async startSession(spec: SessionSpec): Promise<{ sessionId: string; reused: boolean; createdAt: Date }> {
    // Idempotency: Check for in-flight session creation with same execution_id
    // This prevents duplicate sessions when concurrent requests arrive
    const pendingCreation = this.pendingSessionCreations.get(spec.execution_id);
    if (pendingCreation) {
      logger.debug(scopedLog(LogContext.SESSION, 'awaiting in-flight creation'), {
        executionId: spec.execution_id,
        hint: 'Concurrent request detected, waiting for first request to complete',
      });
      return pendingCreation;
    }

    // Create the session creation promise and track it
    const creationPromise = this.startSessionInternal(spec);
    this.pendingSessionCreations.set(spec.execution_id, creationPromise);

    try {
      const result = await creationPromise;
      return result;
    } finally {
      // Always clean up the pending creation tracking
      this.pendingSessionCreations.delete(spec.execution_id);
    }
  }

  /**
   * Internal session creation logic.
   * Separated from startSession to enable in-flight tracking.
   */
  private async startSessionInternal(spec: SessionSpec): Promise<{ sessionId: string; reused: boolean; createdAt: Date }> {
    // Check resource limits
    if (this.sessions.size >= this.config.session.maxConcurrent) {
      logger.warn(scopedLog(LogContext.SESSION, 'resource limit reached'), {
        maxSessions: this.config.session.maxConcurrent,
        currentSessions: this.sessions.size,
        hint: 'Close unused sessions or increase MAX_SESSIONS configuration',
      });
      throw new ResourceLimitError(
        `Maximum concurrent sessions reached: ${this.config.session.maxConcurrent}`,
        { maxSessions: this.config.session.maxConcurrent, currentSessions: this.sessions.size }
      );
    }

    // Idempotency: Check for existing session with same execution_id
    // This handles replay scenarios where the same request is retried
    const existingByExecutionId = this.findSessionByExecutionId(spec.execution_id);
    if (existingByExecutionId) {
      logger.info(scopedLog(LogContext.SESSION, 'idempotent return of existing session'), {
        sessionId: existingByExecutionId.id,
        executionId: spec.execution_id,
        phase: existingByExecutionId.phase,
        hint: 'Request appears to be a retry; returning existing session',
      });

      // For clean mode, reset the session state
      if (spec.reuse_mode === 'clean') {
        await this.resetSession(existingByExecutionId.id);
      }

      existingByExecutionId.lastUsedAt = new Date();
      if (existingByExecutionId.phase !== 'executing') {
        existingByExecutionId.phase = 'ready';
      }
      metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
      return { sessionId: existingByExecutionId.id, reused: true, createdAt: existingByExecutionId.createdAt };
    }

    // Handle reuse mode (match by labels or other criteria)
    if (spec.reuse_mode !== 'fresh') {
      const existingSession = this.findReusableSession(spec);
      if (existingSession) {
        logger.info(scopedLog(LogContext.SESSION, 'reusing existing'), {
          sessionId: existingSession.id,
          reuseMode: spec.reuse_mode,
          previousPhase: existingSession.phase,
          instructionCount: existingSession.instructionCount,
        });

        if (spec.reuse_mode === 'clean') {
          await this.resetSession(existingSession.id);
        }

        existingSession.lastUsedAt = new Date();
        existingSession.phase = 'ready';
        metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
        return { sessionId: existingSession.id, reused: true, createdAt: existingSession.createdAt };
      }
    }

    // Create new session
    const sessionId = uuidv4();
    const createdAt = new Date();

    logger.info(scopedLog(LogContext.SESSION, 'initializing'), {
      sessionId,
      executionId: spec.execution_id,
      reuseMode: spec.reuse_mode,
      viewport: spec.viewport,
    });

    const browser = await this.getBrowser();

    // Build context
    const { context, harPath, tracePath, videoDir } = await buildContext(
      browser,
      spec,
      this.config
    );

    // Create initial page
    const page = await context.newPage();

    // Log page errors (warn level - these are important signals for debugging)
    page.on('pageerror', (err) => {
      logger.warn(scopedLog(LogContext.BROWSER, 'page error'), {
        sessionId,
        error: err.message,
        hint: 'Check the page JavaScript for errors that may affect automation',
      });
    });

    // Log console errors (warn level - only errors, not all console output)
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        logger.warn(scopedLog(LogContext.BROWSER, 'console error'), {
          sessionId,
          text: msg.text(),
        });
      }
    });

    // Network events are collected by telemetry, not logged individually
    // (reduces noise while still capturing data for debugging)

    // Create session state
    const session: SessionState = {
      id: sessionId,
      browser,
      context,
      page,
      spec,
      createdAt,
      lastUsedAt: new Date(),
      tracing: !!tracePath,
      video: !!videoDir,
      harPath,
      tracePath,
      videoDir,
      phase: 'ready',
      instructionCount: 0,
      frameStack: [],
      pages: [page],
      currentPageIndex: 0,
      activeMocks: new Map(),
      // Idempotency: Track executed instructions for replay safety
      executedInstructions: new Map(),
    };

    this.sessions.set(sessionId, session);

    logger.info(scopedLog(LogContext.SESSION, 'ready'), {
      sessionId,
      executionId: spec.execution_id,
      phase: 'ready',
      totalSessions: this.sessions.size,
      viewport: spec.viewport,
    });

    // Update metrics
    metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
    metrics.sessionCount.set({ state: 'total' }, this.sessions.size);

    return { sessionId, reused: false, createdAt };
  }

  /**
   * Get session by ID
   */
  getSession(sessionId: string): SessionState {
    const session = this.sessions.get(sessionId);
    if (!session) {
      throw new SessionNotFoundError(sessionId);
    }

    session.lastUsedAt = new Date();
    return session;
  }

  /**
   * Export the current storage state (cookies/localStorage/etc) for a session.
   */
  async getStorageState(sessionId: string): Promise<unknown> {
    const session = this.getSession(sessionId);
    return session.context.storageState();
  }

  /**
   * Update session activity timestamp without retrieving full session
   * Silently ignores non-existent sessions
   */
  updateActivity(sessionId: string): void {
    const session = this.sessions.get(sessionId);
    if (session) {
      session.lastUsedAt = new Date();
    }
  }

  /**
   * Update session phase.
   * Used to track session lifecycle transitions for observability.
   */
  setSessionPhase(sessionId: string, phase: SessionPhase): void {
    const session = this.sessions.get(sessionId);
    if (session) {
      const previousPhase = session.phase;
      session.phase = phase;
      logger.debug(scopedLog(LogContext.SESSION, 'phase transition'), {
        sessionId,
        from: previousPhase,
        to: phase,
      });
    }
  }

  /**
   * Increment instruction count for a session.
   * Called after each instruction execution for metrics tracking.
   */
  incrementInstructionCount(sessionId: string): void {
    const session = this.sessions.get(sessionId);
    if (session) {
      session.instructionCount++;
    }
  }

  /**
   * Get session info for status endpoints.
   * Returns a summary without exposing internal Playwright objects.
   *
   * Hardened assumptions:
   * - session.page is always defined per SessionState type, but we protect against
   *   edge cases where page might have been closed/detached unexpectedly
   * - page.url() can throw if page has navigated to an error state or been closed
   */
  getSessionInfo(sessionId: string): {
    id: string;
    phase: SessionPhase;
    instructionCount: number;
    createdAt: string;
    lastUsedAt: string;
    isRecording: boolean;
    url: string;
  } {
    const session = this.getSession(sessionId);

    // Hardened: page.url() can throw if page is in an error state
    let currentUrl = 'about:blank';
    try {
      if (session.page && !session.page.isClosed()) {
        currentUrl = session.page.url();
      }
    } catch {
      // Page may have crashed or been detached - use fallback
      currentUrl = 'about:blank';
    }

    return {
      id: session.id,
      phase: session.phase,
      instructionCount: session.instructionCount,
      createdAt: session.createdAt.toISOString(),
      lastUsedAt: session.lastUsedAt.toISOString(),
      isRecording: session.recordingController?.isRecording() ?? false,
      url: currentUrl,
    };
  }

  /**
   * Reset session (navigate to about:blank, clear state)
   */
  async resetSession(sessionId: string): Promise<void> {
    const session = this.getSession(sessionId);
    const previousPhase = session.phase;

    session.phase = 'resetting';
    logger.info(scopedLog(LogContext.SESSION, 'resetting'), {
      sessionId,
      previousPhase,
      instructionCount: session.instructionCount,
    });

    // Navigate to blank page
    await session.page.goto('about:blank');

    // Clear cookies and permissions
    await session.context.clearCookies();
    await session.context.clearPermissions();

    // Clear storage
    await session.page.evaluate(() => {
      // @ts-expect-error - window is available in browser context
      window.localStorage.clear();
      // @ts-expect-error - window is available in browser context
      window.sessionStorage.clear();
    });

    // Reset frame stack
    session.frameStack = [];

    // Close extra pages (tabs)
    if (session.pages.length > 1) {
      const extraPages = session.pages.slice(1);
      for (const page of extraPages) {
        await page.close().catch((err) => {
          logger.warn(scopedLog(LogContext.CLEANUP, 'page close failed'), {
            sessionId,
            error: err.message,
          });
          metrics.cleanupFailures.inc({ operation: 'page_close' });
        });
      }
      session.pages = [session.page];
      session.currentPageIndex = 0;
    }

    // Clear network mocks
    session.activeMocks.clear();

    // Clear executed instructions tracking for fresh state
    session.executedInstructions?.clear();

    session.lastUsedAt = new Date();
    session.phase = 'ready';

    logger.info(scopedLog(LogContext.SESSION, 'reset complete'), {
      sessionId,
      phase: 'ready',
    });
  }

  /**
   * Close session and cleanup resources
   *
   * Hardened to be idempotent - safe to call concurrently from multiple sources
   * (e.g., explicit close and idle cleanup).
   */
  async closeSession(sessionId: string): Promise<void> {
    // Check if session exists
    const session = this.sessions.get(sessionId);
    if (!session) {
      // Session doesn't exist - may have been closed already
      if (this.closingSessionIds.has(sessionId)) {
        // Another call is closing this session - just return
        logger.debug(scopedLog(LogContext.SESSION, 'already closing'), { sessionId });
        return;
      }
      throw new SessionNotFoundError(sessionId);
    }

    // Check if already being closed (concurrent close protection)
    if (this.closingSessionIds.has(sessionId)) {
      logger.debug(scopedLog(LogContext.SESSION, 'close already in progress'), { sessionId });
      return;
    }

    // Mark as closing to prevent concurrent close attempts
    this.closingSessionIds.add(sessionId);

    const previousPhase = session.phase;
    session.phase = 'closing';

    logger.info(scopedLog(LogContext.SESSION, 'closing'), {
      sessionId,
      previousPhase,
      instructionCount: session.instructionCount,
      lifetimeMs: Date.now() - session.createdAt.getTime(),
    });

    const startTime = Date.now();

    try {
      // Stop recording if active
      if (session.recordingController?.isRecording()) {
        await session.recordingController.stopRecording().catch((err) => {
          logger.warn(scopedLog(LogContext.CLEANUP, 'recording stop failed'), {
            sessionId,
            error: err.message,
          });
          metrics.cleanupFailures.inc({ operation: 'recording_stop' });
        });
      }

      // Clean up recording action buffer
      removeRecordedActions(sessionId);

      // Stop tracing if enabled
      if (session.tracing && session.tracePath) {
        await session.context.tracing.stop({ path: session.tracePath }).catch((err) => {
          logger.warn(scopedLog(LogContext.CLEANUP, 'tracing stop failed'), {
            sessionId,
            error: err.message,
          });
          metrics.cleanupFailures.inc({ operation: 'tracing_stop' });
        });
      }

      // Close all pages
      for (const page of session.pages) {
        await page.close().catch((err) => {
          logger.warn(scopedLog(LogContext.CLEANUP, 'page close failed'), {
            sessionId,
            error: err.message,
          });
          metrics.cleanupFailures.inc({ operation: 'page_close' });
        });
      }

      // Close context
      await session.context.close().catch((err) => {
        logger.warn(scopedLog(LogContext.CLEANUP, 'context close failed'), {
          sessionId,
          error: err.message,
        });
        metrics.cleanupFailures.inc({ operation: 'context_close' });
      });

      const duration = Date.now() - startTime;
      metrics.sessionDuration.observe(duration);

      logger.info(scopedLog(LogContext.SESSION, 'closed'), {
        sessionId,
        cleanupDurationMs: duration,
        totalLifetimeMs: Date.now() - session.createdAt.getTime(),
        instructionCount: session.instructionCount,
      });
    } catch (error) {
      logger.error(scopedLog(LogContext.SESSION, 'close failed'), {
        sessionId,
        error: error instanceof Error ? error.message : String(error),
        hint: 'Session cleanup may be incomplete; browser resources may leak',
      });
    } finally {
      this.sessions.delete(sessionId);
      this.closingSessionIds.delete(sessionId);
      metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
      metrics.sessionCount.set({ state: 'total' }, this.sessions.size);
    }
  }

  /**
   * Find session by execution_id.
   * Used for idempotent session lookup when same execution_id is provided.
   */
  private findSessionByExecutionId(executionId: string): SessionState | null {
    for (const session of this.sessions.values()) {
      if (session.spec.execution_id === executionId) {
        return session;
      }
    }
    return null;
  }

  /**
   * Find reusable session matching spec by labels.
   * Note: execution_id matching is now handled separately by findSessionByExecutionId
   * for clearer idempotency semantics.
   */
  private findReusableSession(spec: SessionSpec): SessionState | null {
    for (const session of this.sessions.values()) {
      // Match by labels (if specified)
      if (spec.labels && session.spec.labels) {
        const labelsMatch = Object.entries(spec.labels).every(
          ([key, value]) => session.spec.labels?.[key] === value
        );
        if (labelsMatch) {
          return session;
        }
      }
    }

    return null;
  }

  /**
   * Get count of active sessions
   */
  private getActiveSessionCount(): number {
    const now = Date.now();
    let activeCount = 0;

    for (const session of this.sessions.values()) {
      const idleTimeMs = now - session.lastUsedAt.getTime();
      if (idleTimeMs < this.config.session.idleTimeoutMs) {
        activeCount++;
      }
    }

    return activeCount;
  }

  /**
   * Cleanup idle sessions
   */
  async cleanupIdleSessions(): Promise<void> {
    const now = Date.now();
    const idleSessions: string[] = [];

    for (const [sessionId, session] of this.sessions.entries()) {
      const idleTimeMs = now - session.lastUsedAt.getTime();
      if (idleTimeMs >= this.config.session.idleTimeoutMs) {
        idleSessions.push(sessionId);
      }
    }

    if (idleSessions.length > 0) {
      logger.info('session: cleaning up idle', {
        count: idleSessions.length,
        idleTimeoutMs: this.config.session.idleTimeoutMs,
      });

      for (const sessionId of idleSessions) {
        await this.closeSession(sessionId);
      }

      metrics.sessionCount.set({ state: 'idle' }, 0);
    }
  }

  /**
   * Get all session IDs
   */
  getAllSessionIds(): string[] {
    return Array.from(this.sessions.keys());
  }

  /**
   * Get session count
   */
  getSessionCount(): number {
    return this.sessions.size;
  }

  /**
   * Shutdown manager and cleanup all sessions
   */
  async shutdown(): Promise<void> {
    logger.info('session-manager: shutting down', { sessionCount: this.sessions.size });

    const sessionIds = Array.from(this.sessions.keys());
    for (const sessionId of sessionIds) {
      await this.closeSession(sessionId);
    }

    if (this.browser) {
      await this.browser.close().catch((err) => {
        logger.warn('Failed to close browser', { error: err.message });
        metrics.cleanupFailures.inc({ operation: 'browser_close' });
      });
      this.browser = null;
    }

    logger.info('session-manager: shutdown complete');
  }
}
