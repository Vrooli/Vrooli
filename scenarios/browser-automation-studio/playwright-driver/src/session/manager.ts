import type { SessionSpec, SessionState, SessionPhase, SessionCloseResult } from '../types';
import type { Video } from 'playwright';
import { rename, mkdir } from 'node:fs/promises';
import path from 'node:path';
import type { Config } from '../config';
import { logger, metrics, SessionNotFoundError, ResourceLimitError, scopedLog, LogContext } from '../utils';
import { buildContext } from './context-builder';
import { v4 as uuidv4 } from 'uuid';
import { removeRecordingBuffer } from '../recording/buffer';
import { clearSessionRoutes } from '../handlers/network';
import { clearSessionDownloadCache } from '../handlers/download';
import { BrowserManager, type BrowserStatus } from './browser-manager';
import { transition, canTransition, canAcceptInstructions } from './state-machine';

/**
 * SessionManager - Browser Session Lifecycle Management
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ SESSION LIFECYCLE:                                                      │
 * │                                                                         │
 * │   startSession() ──▶ ready ──▶ executing ──▶ ready ──▶ closeSession()  │
 * │        │                │           │                        │          │
 * │        │                │           │                        │          │
 * │        ▼                ▼           ▼                        ▼          │
 * │   Browser launch   Recording    Instruction          Context close     │
 * │   Context create   if enabled   execution            Browser cleanup   │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * KEY RESPONSIBILITIES:
 * - Session CRUD (create, read, update, delete)
 * - Resource limits (max concurrent sessions)
 * - Idle timeout cleanup
 * - Browser process management (delegated to BrowserManager)
 *
 * IDEMPOTENCY GUARANTEES:
 * - startSession with same execution_id returns existing session (safe for retries)
 * - closeSession can be called multiple times safely
 * - Concurrent session creation with same execution_id deduplicates
 *
 * CONCURRENCY SAFETY:
 * - closeSession may be called from multiple sources (idle cleanup, explicit close)
 * - Uses closingSessionIds Set to prevent double-close
 * - Browser concurrency handled by BrowserManager
 */
export class SessionManager {
  private sessions: Map<string, SessionState> = new Map();
  private browserManager: BrowserManager;
  private config: Config;

  /** Track sessions currently being closed to prevent double-close */
  private closingSessionIds: Set<string> = new Set();

  /**
   * Track in-flight session creation by execution_id.
   * Prevents duplicate session creation when multiple concurrent requests
   * arrive with the same execution_id before the first completes.
   */
  private pendingSessionCreations: Map<string, Promise<{ sessionId: string; reused: boolean; createdAt: Date }>> = new Map();

  constructor(config: Config, browserManager?: BrowserManager) {
    this.config = config;
    this.browserManager = browserManager ?? new BrowserManager(config);
  }

  /**
   * Verify that the browser can be launched.
   * Called during startup to catch Chromium issues early.
   * Returns null on success, error message on failure.
   */
  async verifyBrowserLaunch(): Promise<string | null> {
    return this.browserManager.verifyBrowserLaunch();
  }

  /**
   * Get browser health status for health endpoint.
   */
  getBrowserStatus(): BrowserStatus {
    return this.browserManager.getBrowserStatus();
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

      // Phase recovery: If session is stuck in 'executing' phase (e.g., from a crash
      // or timeout), reset to 'ready' to allow new instructions.
      // This is safe because:
      // 1. We're being called with the same execution_id, indicating a retry
      // 2. The previous execution likely failed or timed out
      // 3. Leaving in 'executing' would permanently block the session
      if (existingByExecutionId.phase === 'executing') {
        logger.warn(scopedLog(LogContext.SESSION, 'recovering from stuck executing phase'), {
          sessionId: existingByExecutionId.id,
          executionId: spec.execution_id,
          previousPhase: 'executing',
          hint: 'Session was stuck in executing phase, resetting to ready',
        });
        existingByExecutionId.phase = 'ready';
      }

      metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
      return { sessionId: existingByExecutionId.id, reused: true, createdAt: existingByExecutionId.createdAt };
    }

    // Handle reuse mode (match by labels or other criteria)
    if (spec.reuse_mode !== 'fresh') {
      const existingSession = this.findReusableSession(spec);
      if (existingSession) {
        // Phase recovery: Log warning if session was stuck in executing phase
        if (existingSession.phase === 'executing') {
          logger.warn(scopedLog(LogContext.SESSION, 'recovering stuck session via reuse'), {
            sessionId: existingSession.id,
            reuseMode: spec.reuse_mode,
            previousPhase: existingSession.phase,
            hint: 'Session was stuck in executing phase, will reset to ready',
          });
        }

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

    const browser = await this.browserManager.getBrowser();

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
   * Update session phase using the state machine.
   *
   * Uses the session state machine to validate transitions.
   * Invalid transitions are logged but don't crash - the phase
   * remains unchanged in that case.
   *
   * @param sessionId - Session ID
   * @param targetPhase - Desired phase to transition to
   * @returns true if transition was successful, false if invalid or session not found
   */
  setSessionPhase(sessionId: string, targetPhase: SessionPhase): boolean {
    const session = this.sessions.get(sessionId);
    if (!session) {
      return false;
    }

    const previousPhase = session.phase;
    const newPhase = transition(previousPhase, targetPhase, sessionId);

    // transition() returns the original phase if invalid
    if (newPhase === previousPhase && newPhase !== targetPhase) {
      // Invalid transition - phase wasn't changed
      return false;
    }

    session.phase = newPhase;
    return true;
  }

  /**
   * Check if a session can accept new instructions.
   *
   * @param sessionId - Session ID
   * @returns true if session exists and can accept instructions
   */
  canAcceptInstructions(sessionId: string): boolean {
    const session = this.sessions.get(sessionId);
    if (!session) {
      return false;
    }
    return canAcceptInstructions(session.phase);
  }

  /**
   * Check if a session can transition to a target phase.
   *
   * @param sessionId - Session ID
   * @param targetPhase - Phase to check transition to
   * @returns true if the transition would be valid
   */
  canTransitionTo(sessionId: string, targetPhase: SessionPhase): boolean {
    const session = this.sessions.get(sessionId);
    if (!session) {
      return false;
    }
    return canTransition(session.phase, targetPhase);
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

    // Clear network mocks from session state
    session.activeMocks.clear();

    // Clear network route tracking (idempotency tracking for network handlers)
    // This ensures replayed network-mock instructions don't incorrectly think
    // routes are already registered after a reset
    clearSessionRoutes(sessionId);

    // Unroute all Playwright route handlers to match cleared state
    await session.page.unroute('**/*').catch((err) => {
      logger.warn(scopedLog(LogContext.CLEANUP, 'unroute failed'), {
        sessionId,
        error: err.message,
      });
    });

    // Clear download cache (idempotency tracking for download handlers)
    // This ensures replayed download instructions will actually download again
    clearSessionDownloadCache(sessionId);

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
  async closeSession(sessionId: string): Promise<SessionCloseResult> {
    // Check if session exists
    const session = this.sessions.get(sessionId);
    if (!session) {
      // Session doesn't exist - may have been closed already
      if (this.closingSessionIds.has(sessionId)) {
        // Another call is closing this session - just return
        logger.debug(scopedLog(LogContext.SESSION, 'already closing'), { sessionId });
        return { videoPaths: [] };
      }
      throw new SessionNotFoundError(sessionId);
    }

    // Check if already being closed (concurrent close protection)
    if (this.closingSessionIds.has(sessionId)) {
      logger.debug(scopedLog(LogContext.SESSION, 'close already in progress'), { sessionId });
      return { videoPaths: [] };
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

    let videoPaths: string[] = [];
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

      // Clean up recording buffer
      removeRecordingBuffer(sessionId);

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

      // Close all pages and collect video artifacts (if enabled)
      for (const [index, page] of session.pages.entries()) {
        const video = page.video();
        await page.close().catch((err) => {
          logger.warn(scopedLog(LogContext.CLEANUP, 'page close failed'), {
            sessionId,
            error: err.message,
          });
          metrics.cleanupFailures.inc({ operation: 'page_close' });
        });
        if (video) {
          const videoPath = await resolveVideoPath(video, session, index);
          if (videoPath) {
            videoPaths.push(videoPath);
          }
        }
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
    return { videoPaths };
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

    await this.browserManager.shutdown();

    logger.info('session-manager: shutdown complete');
  }
}

async function resolveVideoPath(video: Video, session: SessionState, pageIndex: number): Promise<string | null> {
  let sourcePath = '';
  try {
    sourcePath = await video.path();
  } catch (error) {
    logger.warn(scopedLog(LogContext.CLEANUP, 'video path unavailable'), {
      sessionId: session.id,
      error: error instanceof Error ? error.message : String(error),
    });
    return null;
  }

  if (!sourcePath) {
    return null;
  }

  const ext = path.extname(sourcePath) || '.webm';
  const targetDir = session.videoDir || path.dirname(sourcePath);
  const targetName = `execution-${session.spec.execution_id}-page-${pageIndex + 1}${ext}`;
  const targetPath = path.join(targetDir, targetName);

  if (targetPath === sourcePath) {
    return sourcePath;
  }

  try {
    await mkdir(targetDir, { recursive: true });
    await rename(sourcePath, targetPath);
    return targetPath;
  } catch (error) {
    logger.warn(scopedLog(LogContext.CLEANUP, 'video rename failed'), {
      sessionId: session.id,
      sourcePath,
      targetPath,
      error: error instanceof Error ? error.message : String(error),
    });
    return sourcePath;
  }
}
