import { chromium, Browser } from 'playwright';
import type { SessionSpec, SessionState } from '../types';
import type { Config } from '../config';
import { logger, metrics, SessionNotFoundError, ResourceLimitError } from '../utils';
import { buildContext } from './context-builder';
import { v4 as uuidv4 } from 'uuid';
import { cleanupSessionRecording } from '../routes/record-mode';

/**
 * SessionManager - Manages browser session lifecycle
 *
 * Responsibilities:
 * - Session creation and deletion
 * - Session retrieval by ID
 * - Resource limit enforcement
 * - Idle timeout tracking
 * - Browser process management
 */
export class SessionManager {
  private sessions: Map<string, SessionState> = new Map();
  private browser: Browser | null = null;
  private config: Config;

  private browserVerified = false;
  private browserError: string | null = null;

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
      logger.info('Verifying browser launch capability...');
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

      logger.info('Browser launch verification successful', {
        version: browser.version(),
      });

      return null;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.browserError = errorMessage;
      this.browserVerified = true; // Mark as verified (we checked, it failed)

      logger.error('Browser launch verification failed', {
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
   * Get or create shared browser instance
   */
  private async getBrowser(): Promise<Browser> {
    if (this.browser && this.browser.isConnected()) {
      return this.browser;
    }

    logger.info('Launching browser', {
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || 'auto',
    });

    this.browser = await chromium.launch({
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || undefined,
      args: this.config.browser.args,
    });

    logger.info('Browser launched', { version: this.browser.version() });
    return this.browser;
  }

  /**
   * Start a new session
   */
  async startSession(spec: SessionSpec): Promise<string> {
    // Check resource limits
    if (this.sessions.size >= this.config.session.maxConcurrent) {
      throw new ResourceLimitError(
        `Maximum concurrent sessions reached: ${this.config.session.maxConcurrent}`,
        { maxSessions: this.config.session.maxConcurrent, currentSessions: this.sessions.size }
      );
    }

    // Handle reuse mode
    if (spec.reuse_mode !== 'fresh') {
      const existingSession = this.findReusableSession(spec);
      if (existingSession) {
        logger.info('Reusing existing session', {
          sessionId: existingSession.id,
          reuseMode: spec.reuse_mode,
        });

        if (spec.reuse_mode === 'clean') {
          await this.resetSession(existingSession.id);
        }

        existingSession.lastUsedAt = new Date();
        metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
        return existingSession.id;
      }
    }

    // Create new session
    const sessionId = uuidv4();
    const browser = await this.getBrowser();

    // Build context
    const { context, harPath, tracePath, videoDir } = await buildContext(
      browser,
      spec,
      this.config
    );

    // Create initial page
    const page = await context.newPage();

    // Setup console log collection
    page.on('console', (msg) => {
      logger.debug('Browser console', {
        sessionId,
        type: msg.type(),
        text: msg.text(),
      });
    });

    // Setup network event collection
    page.on('request', (request) => {
      logger.debug('Network request', {
        sessionId,
        url: request.url(),
        method: request.method(),
      });
    });

    page.on('response', (response) => {
      logger.debug('Network response', {
        sessionId,
        url: response.url(),
        status: response.status(),
      });
    });

    // Create session state
    const session: SessionState = {
      id: sessionId,
      browser,
      context,
      page,
      spec,
      createdAt: new Date(),
      lastUsedAt: new Date(),
      tracing: !!tracePath,
      video: !!videoDir,
      harPath,
      tracePath,
      videoDir,
      frameStack: [],
      pages: [page],
      currentPageIndex: 0,
      activeMocks: new Map(),
    };

    this.sessions.set(sessionId, session);

    logger.info('Session created', {
      sessionId,
      executionId: spec.execution_id,
      reuseMode: spec.reuse_mode,
      totalSessions: this.sessions.size,
    });

    // Update metrics
    metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
    metrics.sessionCount.set({ state: 'total' }, this.sessions.size);

    return sessionId;
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
   * Reset session (navigate to about:blank, clear state)
   */
  async resetSession(sessionId: string): Promise<void> {
    const session = this.getSession(sessionId);

    logger.info('Resetting session', { sessionId });

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
          logger.warn('Failed to close page during reset', { sessionId, error: err.message });
          metrics.cleanupFailures.inc({ operation: 'page_close' });
        });
      }
      session.pages = [session.page];
      session.currentPageIndex = 0;
    }

    // Clear network mocks
    session.activeMocks.clear();

    session.lastUsedAt = new Date();

    logger.info('Session reset complete', { sessionId });
  }

  /**
   * Close session and cleanup resources
   */
  async closeSession(sessionId: string): Promise<void> {
    const session = this.sessions.get(sessionId);
    if (!session) {
      throw new SessionNotFoundError(sessionId);
    }

    logger.info('Closing session', { sessionId });

    const startTime = Date.now();

    try {
      // Stop recording if active
      if (session.recordingController?.isRecording()) {
        await session.recordingController.stopRecording().catch((err) => {
          logger.warn('Failed to stop recording during session close', { sessionId, error: err.message });
          metrics.cleanupFailures.inc({ operation: 'recording_stop' });
        });
      }

      // Clean up recording action buffer
      cleanupSessionRecording(sessionId);

      // Stop tracing if enabled
      if (session.tracing && session.tracePath) {
        await session.context.tracing.stop({ path: session.tracePath }).catch((err) => {
          logger.warn('Failed to stop tracing', { sessionId, error: err.message });
          metrics.cleanupFailures.inc({ operation: 'tracing_stop' });
        });
      }

      // Close all pages
      for (const page of session.pages) {
        await page.close().catch((err) => {
          logger.warn('Failed to close page', { sessionId, error: err.message });
          metrics.cleanupFailures.inc({ operation: 'page_close' });
        });
      }

      // Close context
      await session.context.close().catch((err) => {
        logger.warn('Failed to close context', { sessionId, error: err.message });
        metrics.cleanupFailures.inc({ operation: 'context_close' });
      });

      const duration = Date.now() - startTime;
      metrics.sessionDuration.observe(duration);

      logger.info('Session closed', {
        sessionId,
        duration,
        createdAt: session.createdAt,
        lastUsedAt: session.lastUsedAt,
      });
    } catch (error) {
      logger.error('Error closing session', {
        sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    } finally {
      this.sessions.delete(sessionId);
      metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
      metrics.sessionCount.set({ state: 'total' }, this.sessions.size);
    }
  }

  /**
   * Find reusable session matching spec
   */
  private findReusableSession(spec: SessionSpec): SessionState | null {
    for (const session of this.sessions.values()) {
      // Match by execution_id
      if (session.spec.execution_id === spec.execution_id) {
        return session;
      }

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
      logger.info('Cleaning up idle sessions', {
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
    logger.info('Shutting down session manager', { sessionCount: this.sessions.size });

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

    logger.info('Session manager shutdown complete');
  }
}
