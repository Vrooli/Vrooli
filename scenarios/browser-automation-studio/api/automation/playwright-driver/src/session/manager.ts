import { chromium, Browser, Page } from 'playwright';
import type { SessionSpec, SessionState } from '../types';
import type { Config } from '../config';
import { logger, metrics, SessionNotFoundError, ResourceLimitError } from '../utils';
import { buildContext } from './context-builder';
import { v4 as uuidv4 } from 'uuid';

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

  constructor(config: Config) {
    this.config = config;
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
   * Reset session (navigate to about:blank, clear state)
   */
  async resetSession(sessionId: string): Promise<void> {
    const session = this.getSession(sessionId);

    logger.info('Resetting session', { sessionId });

    // Navigate to blank page
    await session.page.goto('about:blank');

    // Clear cookies
    await session.context.clearCookies();

    // Clear storage
    await session.page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });

    // Reset frame stack
    session.frameStack = [];

    // Close extra pages (tabs)
    if (session.pages.length > 1) {
      const extraPages = session.pages.slice(1);
      for (const page of extraPages) {
        await page.close().catch((err) => {
          logger.warn('Failed to close page during reset', { sessionId, error: err.message });
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
      logger.warn('Attempted to close non-existent session', { sessionId });
      return;
    }

    logger.info('Closing session', { sessionId });

    const startTime = Date.now();

    try {
      // Stop tracing if enabled
      if (session.tracing && session.tracePath) {
        await session.context.tracing.stop({ path: session.tracePath }).catch((err) => {
          logger.warn('Failed to stop tracing', { sessionId, error: err.message });
        });
      }

      // Close all pages
      for (const page of session.pages) {
        await page.close().catch((err) => {
          logger.warn('Failed to close page', { sessionId, error: err.message });
        });
      }

      // Close context
      await session.context.close().catch((err) => {
        logger.warn('Failed to close context', { sessionId, error: err.message });
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
      });
      this.browser = null;
    }

    logger.info('Session manager shutdown complete');
  }
}
