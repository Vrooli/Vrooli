import type { SessionSpec, SessionState, SessionPhase, SessionCloseResult } from '../types';
import type { Video } from 'rebrowser-playwright';
import { rename, mkdir } from 'node:fs/promises';
import path from 'node:path';
import type { Config } from '../config';
import { logger, metrics, SessionNotFoundError, ResourceLimitError, scopedLog, LogContext } from '../utils';
import { buildContext, type ActualViewport } from './context-builder';
import { v4 as uuidv4 } from 'uuid';
import { removeRecordingBuffer, RecordingPipelineManager } from '../recording';
import { cleanupSession, createInFlightGuard, type InFlightGuard } from '../infra';
import { BrowserManager, type BrowserStatus } from './browser-manager';
import { transition, canTransition, canAcceptInstructions } from './state-machine';
import {
  findByExecutionId,
  findByLabels,
  shouldAttemptReuse,
  makeReuseDecision,
  findIdleSessions,
} from './session-decisions';
import { setupDiagnosticLogging } from './diagnostic-logger';

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
/** Result type for session creation */
type SessionCreationResult = { sessionId: string; reused: boolean; createdAt: Date; actualViewport: ActualViewport };

export class SessionManager {
  private sessions: Map<string, SessionState> = new Map();
  private browserManager: BrowserManager;
  private config: Config;

  /** Track sessions currently being closed to prevent double-close */
  private closingSessionIds: Set<string> = new Set();

  /**
   * In-flight guard for session creation.
   * Prevents duplicate session creation when multiple concurrent requests
   * arrive with the same execution_id before the first completes.
   */
  private sessionCreationGuard: InFlightGuard<string, SessionCreationResult>;

  constructor(config: Config, browserManager?: BrowserManager) {
    this.config = config;
    this.browserManager = browserManager ?? new BrowserManager(config);
    this.sessionCreationGuard = createInFlightGuard<string, SessionCreationResult>({
      name: 'session-creation',
      logContext: LogContext.SESSION,
    });
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
   * - Uses InFlightGuard to prevent race conditions under concurrent requests
   *
   * @returns Object with session ID, whether it was reused, and the actual viewport with source attribution
   */
  async startSession(spec: SessionSpec): Promise<SessionCreationResult> {
    // InFlightGuard handles concurrent request deduplication automatically
    return this.sessionCreationGuard.execute(
      spec.execution_id,
      () => this.startSessionInternal(spec)
    );
  }

  /**
   * Internal session creation logic.
   * Separated from startSession to enable InFlightGuard tracking.
   */
  private async startSessionInternal(spec: SessionSpec): Promise<SessionCreationResult> {
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
    // Decision logic is in session-decisions.ts
    const existingByExecutionId = findByExecutionId(this.sessions.values(), spec.execution_id);
    if (existingByExecutionId) {
      const decision = makeReuseDecision(existingByExecutionId, spec, 'execution_id_match');

      logger.info(scopedLog(LogContext.SESSION, 'idempotent return of existing session'), {
        sessionId: existingByExecutionId.id,
        executionId: spec.execution_id,
        phase: existingByExecutionId.phase,
        decision: decision.reason,
      });

      // Apply reuse decision
      if (decision.shouldReset) {
        await this.resetSession(existingByExecutionId.id);
      }

      existingByExecutionId.lastUsedAt = new Date();

      // Phase recovery based on decision
      if (decision.shouldRecoverPhase) {
        logger.warn(scopedLog(LogContext.SESSION, 'recovering from stuck executing phase'), {
          sessionId: existingByExecutionId.id,
          executionId: spec.execution_id,
          previousPhase: 'executing',
          hint: decision.reason,
        });
        existingByExecutionId.phase = 'ready';
      }

      metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
      const viewportSize = existingByExecutionId.page.viewportSize() ?? { width: 1280, height: 720 };
      const actualViewport: ActualViewport = {
        width: viewportSize.width,
        height: viewportSize.height,
        source: 'requested', // Reused session - original source unknown
        reason: 'Reused existing session',
      };
      return { sessionId: existingByExecutionId.id, reused: true, createdAt: existingByExecutionId.createdAt, actualViewport };
    }

    // Handle reuse mode (match by labels)
    // Decision logic is in session-decisions.ts
    if (shouldAttemptReuse(spec.reuse_mode)) {
      const existingSession = findByLabels(this.sessions.values(), spec.labels);
      if (existingSession) {
        const decision = makeReuseDecision(existingSession, spec, 'label_match');

        // Log warning if session was stuck in executing phase
        if (decision.shouldRecoverPhase) {
          logger.warn(scopedLog(LogContext.SESSION, 'recovering stuck session via reuse'), {
            sessionId: existingSession.id,
            reuseMode: spec.reuse_mode,
            previousPhase: existingSession.phase,
            hint: decision.reason,
          });
        }

        logger.info(scopedLog(LogContext.SESSION, 'reusing existing'), {
          sessionId: existingSession.id,
          reuseMode: spec.reuse_mode,
          previousPhase: existingSession.phase,
          instructionCount: existingSession.instructionCount,
          decision: decision.reason,
        });

        if (decision.shouldReset) {
          await this.resetSession(existingSession.id);
        }

        existingSession.lastUsedAt = new Date();
        existingSession.phase = 'ready';
        metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
        const viewportSize = existingSession.page.viewportSize() ?? { width: 1280, height: 720 };
        const actualViewport: ActualViewport = {
          width: viewportSize.width,
          height: viewportSize.height,
          source: 'requested', // Reused session - original source unknown
          reason: 'Reused existing session by label match',
        };
        return { sessionId: existingSession.id, reused: true, createdAt: existingSession.createdAt, actualViewport };
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

    // Build context (includes actualViewport with source attribution)
    const { context, harPath, tracePath, videoDir, serviceWorkerController, recordingInitializer, actualViewport } = await buildContext(
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

    // Create recording pipeline manager (eager instantiation)
    // This allows early verification and ensures the pipeline is ready before recording starts
    const pipelineManager = new RecordingPipelineManager(page, context, recordingInitializer, {
      sessionId,
      logger,
    });

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
      pageIdMap: new Map(),
      pageToIdMap: new WeakMap(),
      activeMocks: new Map(),
      // Idempotency: Track executed instructions for replay safety
      executedInstructions: new Map(),
      // Service worker control
      serviceWorkerController,
      // Recording context initializer (binding + init script)
      recordingInitializer,
      // Recording pipeline manager (single source of truth for recording state)
      pipelineManager,
    };

    // Assign an ID to the initial page and track it
    const initialPageId = crypto.randomUUID();
    session.pageIdMap.set(initialPageId, page);
    session.pageToIdMap.set(page, initialPageId);

    this.sessions.set(sessionId, session);

    // Setup diagnostic logging for redirect loop debugging
    // Enable with DIAGNOSTIC_LOGGING=true environment variable
    setupDiagnosticLogging(context, sessionId);

    // Initialize recording pipeline (early verification)
    // This runs injection and verification so the pipeline is ready before recording starts
    // The promise is stored in session.pipelineReadyPromise so consumers can await it
    const pipelineReadyPromise = pipelineManager.initialize().then(() => {
      return pipelineManager.verifyPipeline({ timeoutMs: 5000, retries: 1 });
    }).then((verification) => {
      if (verification.scriptLoaded && verification.scriptReady && verification.inMainContext) {
        logger.debug(scopedLog(LogContext.SESSION, 'recording pipeline verified'), {
          sessionId,
          handlersCount: verification.handlersCount,
        });
        return true;
      } else {
        logger.warn(scopedLog(LogContext.SESSION, 'recording pipeline verification incomplete'), {
          sessionId,
          verification,
          hint: 'Recording may require re-verification on first use',
        });
        return false;
      }
    }).catch((err) => {
      logger.warn(scopedLog(LogContext.SESSION, 'recording pipeline init failed'), {
        sessionId,
        error: err instanceof Error ? err.message : String(err),
        hint: 'Recording will retry initialization when started',
      });
      return false;
    });

    // Store the promise in session state for consumers to await
    session.pipelineReadyPromise = pipelineReadyPromise;

    // Enable service worker monitoring and handle unregisterOnStart
    await serviceWorkerController.enable(page);
    const swControl = spec.service_worker_control;
    if (swControl?.unregisterOnStart || swControl?.mode === 'unregister-all') {
      const unregisteredCount = await serviceWorkerController.unregisterAll();
      if (unregisteredCount > 0) {
        logger.debug(scopedLog(LogContext.SESSION, 'SWs unregistered on start'), {
          sessionId,
          count: unregisteredCount,
        });
      }
    }

    logger.info(scopedLog(LogContext.SESSION, 'ready'), {
      sessionId,
      executionId: spec.execution_id,
      phase: 'ready',
      totalSessions: this.sessions.size,
      viewport: spec.viewport,
      initialPageId,
    });

    // Update metrics
    metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
    metrics.sessionCount.set({ state: 'total' }, this.sessions.size);

    // Return actualViewport from buildContext (includes source attribution)
    return { sessionId, reused: false, createdAt, actualViewport };
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
   * Wait for the recording pipeline to be ready.
   *
   * This should be called before starting operations that depend on the pipeline
   * being initialized and verified, such as frame streaming. The pipeline is
   * initialized asynchronously during session creation, so this method allows
   * consumers to wait until it's ready.
   *
   * @param sessionId - Session ID
   * @param timeoutMs - Maximum time to wait (default: 10000ms)
   * @returns true if pipeline is ready, false if verification failed or timeout
   */
  async waitForPipelineReady(sessionId: string, timeoutMs = 10000): Promise<boolean> {
    const session = this.sessions.get(sessionId);
    if (!session) {
      return false;
    }

    // If no pipeline manager, nothing to wait for
    if (!session.pipelineManager) {
      return true;
    }

    // If already ready (phase check), return immediately
    if (session.pipelineManager.isReady()) {
      return true;
    }

    // If we have a readiness promise, wait for it with timeout
    if (session.pipelineReadyPromise) {
      try {
        const result = await Promise.race([
          session.pipelineReadyPromise,
          new Promise<boolean>((resolve) => setTimeout(() => resolve(false), timeoutMs)),
        ]);
        return result;
      } catch {
        return false;
      }
    }

    return false;
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
      isRecording: session.pipelineManager?.isRecording() ?? false,
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

    // Unroute all Playwright route handlers to match cleared state
    await session.page.unroute('**/*').catch((err) => {
      logger.warn(scopedLog(LogContext.CLEANUP, 'unroute failed'), {
        sessionId,
        error: err.message,
      });
    });

    // Clear handler-specific caches via the cleanup registry
    // This consolidates cleanup operations and eliminates direct handler imports
    await cleanupSession(sessionId);

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
      // Stop recording if active (use pipelineManager as single source of truth)
      if (session.pipelineManager?.isRecording()) {
        await session.pipelineManager.stopRecording().catch((err) => {
          logger.warn(scopedLog(LogContext.CLEANUP, 'recording stop failed'), {
            sessionId,
            error: err instanceof Error ? err.message : String(err),
          });
          metrics.cleanupFailures.inc({ operation: 'recording_stop' });
        });
      }

      // Clean up recording buffer
      removeRecordingBuffer(sessionId);

      // Disable service worker monitoring
      if (session.serviceWorkerController) {
        await session.serviceWorkerController.disable().catch((err) => {
          logger.warn(scopedLog(LogContext.CLEANUP, 'SW controller disable failed'), {
            sessionId,
            error: err instanceof Error ? err.message : String(err),
          });
        });
      }

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
    return {
      videoPaths,
      tracePath: session.tracePath,
      harPath: session.harPath,
    };
  }

  // Session lookup functions moved to session-decisions.ts:
  // - findByExecutionId (was findSessionByExecutionId)
  // - findByLabels (was findReusableSession)
  // - makeReuseDecision (new - encapsulates reuse logic)
  // - findIdleSessions (new - encapsulates idle detection)

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
   * Decision logic is in session-decisions.ts
   */
  async cleanupIdleSessions(): Promise<void> {
    const idleSessions = findIdleSessions(this.sessions, this.config.session.idleTimeoutMs);

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
   * Get a summary of session statistics for observability.
   * Used by the /observability endpoint.
   */
  getSessionSummary(): {
    total: number;
    active: number;
    idle: number;
    active_recordings: number;
    idle_timeout_ms: number;
    capacity: number;
  } {
    const now = Date.now();
    let active = 0;
    let idle = 0;
    let activeRecordings = 0;

    for (const session of this.sessions.values()) {
      const idleTimeMs = now - session.lastUsedAt.getTime();
      if (idleTimeMs < this.config.session.idleTimeoutMs) {
        active++;
      } else {
        idle++;
      }

      if (session.pipelineManager?.isRecording()) {
        activeRecordings++;
      }
    }

    return {
      total: this.sessions.size,
      active,
      idle,
      active_recordings: activeRecordings,
      idle_timeout_ms: this.config.session.idleTimeoutMs,
      capacity: this.config.session.maxConcurrent,
    };
  }

  /**
   * Get detailed list of all sessions for observability/diagnostics.
   * Returns non-sensitive session metadata.
   */
  getSessionList(): Array<{
    id: string;
    phase: SessionPhase;
    created_at: string;
    last_used_at: string;
    idle_time_ms: number;
    is_idle: boolean;
    is_recording: boolean;
    instruction_count: number;
    workflow_id?: string;
    current_url?: string;
    page_count: number;
  }> {
    const now = Date.now();
    const list: Array<{
      id: string;
      phase: SessionPhase;
      created_at: string;
      last_used_at: string;
      idle_time_ms: number;
      is_idle: boolean;
      is_recording: boolean;
      instruction_count: number;
      workflow_id?: string;
      current_url?: string;
      page_count: number;
    }> = [];

    for (const session of this.sessions.values()) {
      const idleTimeMs = now - session.lastUsedAt.getTime();
      let currentUrl: string | undefined;
      try {
        currentUrl = session.page?.url();
      } catch {
        // Page may be closed
      }

      list.push({
        id: session.id,
        phase: session.phase,
        created_at: session.createdAt.toISOString(),
        last_used_at: session.lastUsedAt.toISOString(),
        idle_time_ms: idleTimeMs,
        is_idle: idleTimeMs >= this.config.session.idleTimeoutMs,
        is_recording: session.pipelineManager?.isRecording() ?? false,
        instruction_count: session.instructionCount,
        workflow_id: session.spec.workflow_id,
        current_url: currentUrl,
        page_count: session.pages.length,
      });
    }

    // Sort by last used (most recent first)
    return list.sort((a, b) => new Date(b.last_used_at).getTime() - new Date(a.last_used_at).getTime());
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
