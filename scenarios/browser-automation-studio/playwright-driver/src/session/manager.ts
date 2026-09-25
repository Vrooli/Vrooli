import type {
  SessionSpec,
  SessionState,
  SessionPhase,
  SessionCloseResult,
  AppTargetSpec,
} from '../types';
import type { Browser, BrowserContext, Page } from 'rebrowser-playwright';
import path from 'node:path';
import type { Config } from '../config';
import {
  logger,
  metrics,
  SessionNotFoundError,
  ResourceLimitError,
  scopedLog,
  LogContext,
} from '../utils';
import { buildContext, type ActualViewport } from './context-builder';
import { v4 as uuidv4 } from 'uuid';
import { RecordingPipelineManager, createRecordingContextInitializer } from '../recording';
import { ServiceWorkerController } from '../service-worker';
import { createInFlightGuard, type InFlightGuard } from '../infra';
import { BrowserManager, type BrowserStatus } from './browser-manager';
import { applySilentSinkToCurrentPage, generateSilentSinkPatch, type AudioStrategy } from './audio';
import {
  createPipeWireQualificationDevice,
  PIPEWIRE_QUALIFICATION_DEVICE_NAME,
  verifyBrowserCaptureDevice,
  type BrowserCaptureDeviceEvidence,
  type PipeWireQualificationDevice,
} from './audio/device-evidence';
import { transition, canTransition, canAcceptInstructions } from './state-machine';
import {
  findByExecutionId,
  findByLabels,
  shouldAttemptReuse,
  findIdleSessions,
} from './session-decisions';
import { setupDiagnosticLogging } from './diagnostic-logger';
import { resolveInstrumentation, safeInvoke, type Instrumentation } from '../instrumentation';
import { PerformanceTracer, injectWebVitalsObserver, AccessibilitySnapshotter } from '../tracing';
import {
  countActiveSessions,
  inspectSession,
  listSessions,
  summarizeSessions,
  type SessionInfo,
  type SessionListEntry,
  type SessionSummary,
} from './session-inspection';
import { resetSessionState } from './session-reset';
import { teardownSessionResources } from './session-teardown';
import {
  selectAppTargetPage,
  validateAppTargetCapabilities,
  validateAppTargetSpec,
  verifyAppTargetRenderer,
} from './electron-target';

/** Keep workflow tab operations aligned with pages created outside the tab handler. */
function trackSessionPageStack(session: SessionState): void {
  const trackPage = (page: Page): void => {
    if (!session.pages.includes(page)) session.pages.push(page);
    page.once('close', () => {
      const index = session.pages.indexOf(page);
      if (index !== -1) session.pages.splice(index, 1);
      if (session.page === page) {
        session.frameStack.length = 0;
        const next = session.pages.find((candidate) => !candidate.isClosed());
        if (next) session.page = next;
      }
      session.currentPageIndex = session.pages.indexOf(session.page);
    });
  };

  session.context.on('page', trackPage);
  session.pages.forEach(trackPage);
}

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
 * - Concurrent close callers share the same pending result
 * - Browser concurrency handled by BrowserManager
 */
/** Result type for session creation */
type SessionCreationResult = {
  sessionId: string;
  leaseId: string;
  reused: boolean;
  createdAt: Date;
  actualViewport: ActualViewport;
};

const getErrorMessage = (error: unknown): string =>
  error instanceof Error ? error.message : String(error);

export class SessionManager {
  private sessions: Map<string, SessionState> = new Map();
  /** New sessions that reserved capacity but are not in the session map yet. */
  private reservedSessionStarts = 0;
  private browserManager: BrowserManager;
  private config: Config;
  private qualificationDevice: Promise<PipeWireQualificationDevice> | null = null;
  /** Number of device-evidence sessions still being admitted before map insertion. */
  private qualificationDeviceStarts = 0;

  /**
   * Cross-cutting instrumentation seam (no-op by default). Session-level
   * hooks fire when a session becomes ready and when it is closed. P2
   * supplies a real implementation; P1 keeps it inert.
   */
  private instrumentation: Instrumentation;

  private resettingSessions = new Map<string, Promise<void>>();

  /** Track sessions currently being closed to prevent double-close */
  private closingSessions = new Map<string, { session: SessionState; result: Promise<SessionCloseResult> }>();

  /**
   * In-flight guard for session creation.
   * Prevents duplicate session creation when multiple concurrent requests
   * arrive with the same execution_id before the first completes.
   */
  private sessionCreationGuard: InFlightGuard<string, SessionCreationResult>;

  constructor(config: Config, browserManager?: BrowserManager, instrumentation?: Instrumentation) {
    this.config = config;
    this.browserManager = browserManager ?? new BrowserManager(config);
    this.instrumentation = resolveInstrumentation(instrumentation);
    this.sessionCreationGuard = createInFlightGuard<string, SessionCreationResult>({
      name: 'session-creation',
      logContext: LogContext.SESSION,
    });
  }

  /**
   * Returns the instrumentation seam. Exposed so the route layer can
   * thread the same instance into per-instruction execution.
   */
  getInstrumentation(): Instrumentation {
    return this.instrumentation;
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

  /** Close the shared qualification topology when no session owns it. */
  private async closeQualificationDeviceIfIdle(): Promise<void> {
    if (
      this.sessions.size !== 0 ||
      this.qualificationDeviceStarts !== 0 ||
      !this.qualificationDevice
    )
      return;
    const qualificationDevice = this.qualificationDevice;
    this.qualificationDevice = null;
    await qualificationDevice
      .then((device) => device.close())
      .catch((error: unknown) =>
        logger.warn(scopedLog(LogContext.CLEANUP, 'qualification device cleanup failed'), {
          error: getErrorMessage(error),
        })
      );
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
    return this.sessionCreationGuard.execute(spec.execution_id, () =>
      this.startSessionInternal(spec)
    );
  }

  /**
   * Internal session creation logic.
   * Separated from startSession to enable InFlightGuard tracking.
   */
  private async startSessionInternal(spec: SessionSpec): Promise<SessionCreationResult> {
    let reserved = false;
    try {
      return await this.startSessionAttempt(spec, () => {
        this.reservedSessionStarts += 1;
        reserved = true;
      });
    } finally {
      if (reserved) this.reservedSessionStarts -= 1;
    }
  }

  /**
   * Internal session creation logic. Reservation is acquired synchronously at
   * the admission check and released by startSessionInternal on every outcome.
   */
  private async startSessionAttempt(
    spec: SessionSpec,
    reserveCapacity: () => void
  ): Promise<SessionCreationResult> {
    // Idempotency: Check for existing session with same execution_id
    // Decision logic is in session-decisions.ts
    const existingByExecutionId = findByExecutionId(this.sessions.values(), spec.execution_id);
    if (existingByExecutionId) {
      logger.info(scopedLog(LogContext.SESSION, 'idempotent return of existing session'), {
        sessionId: existingByExecutionId.id,
        executionId: spec.execution_id,
        phase: existingByExecutionId.phase,
      });

      // A repeated start observes the same lease. It cannot establish that a
      // pending action was abandoned or authorize clearing browser state.
      existingByExecutionId.lastUsedAt = new Date();

      metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
      const viewportSize = existingByExecutionId.page.viewportSize() ?? {
        width: 1280,
        height: 720,
      };
      const actualViewport: ActualViewport = {
        width: viewportSize.width,
        height: viewportSize.height,
        source: 'requested', // Reused session - original source unknown
        reason: 'Reused existing session',
      };
      return {
        sessionId: existingByExecutionId.id,
        leaseId: existingByExecutionId.leaseId,
        reused: true,
        createdAt: existingByExecutionId.createdAt,
        actualViewport,
      };
    }

    // Handle reuse mode (match by labels)
    // Decision logic is in session-decisions.ts
    if (shouldAttemptReuse(spec.reuse_mode)) {
      const existingSession = findByLabels(this.sessions.values(), spec.labels);
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

        // The previous owner explicitly released this lease. A new execution
        // gets a new immutable ownership token; it never mutates an active
        // execution identity in place.
        existingSession.ownerExecutionId = spec.execution_id;
        existingSession.leaseId = uuidv4();
        existingSession.leaseReleasedAt = undefined;
        existingSession.instructionReceipts?.clear();
        existingSession.lastInstructionSequence = 0;
        existingSession.instructionCount = 0;
        existingSession.spec = {
          ...existingSession.spec,
          ...spec,
        };
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
        return {
          sessionId: existingSession.id,
          leaseId: existingSession.leaseId,
          reused: true,
          createdAt: existingSession.createdAt,
          actualViewport,
        };
      }
    }

    // New sessions consume capacity only after idempotent and released-lease
    // reuse paths have been considered.
    const occupiedSlots = this.sessions.size + this.reservedSessionStarts;
    if (occupiedSlots >= this.config.session.maxConcurrent) {
      logger.warn(scopedLog(LogContext.SESSION, 'resource limit reached'), {
        maxSessions: this.config.session.maxConcurrent,
        currentSessions: this.sessions.size,
        pendingStarts: this.reservedSessionStarts,
        hint: 'Release or close unused sessions, or increase MAX_SESSIONS configuration',
      });
      throw new ResourceLimitError(
        `Maximum concurrent sessions reached: ${this.config.session.maxConcurrent}`,
        {
          maxSessions: this.config.session.maxConcurrent,
          currentSessions: this.sessions.size,
          pendingStarts: this.reservedSessionStarts,
        }
      );
    }
    reserveCapacity();

    if (spec.app_target) {
      return this.startAppTargetSessionInternal(spec, spec.app_target);
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

    const fakeMicrophoneWav = spec.fake_media?.microphone_wav?.trim();
    // Prefer the explicit per-session request. Keep the environment switch as
    // a compatibility seam for the dedicated operator qualification service,
    // but do not make every BAS session mutate the host audio topology just
    // because that process happens to share an environment.
    const deviceEvidenceEnabled =
      spec.audio_device_evidence === true || process.env.VROOLI_AUDIO_DEVICE_EVIDENCE === '1';
    let audioPlaybackStop: (() => Promise<void>) | undefined;
    let audioPlaybackRestart: (() => Promise<void>) | undefined;
    let audioPlaybackFailure: string | undefined;
    let contextForCleanup: BrowserContext | undefined;
    let browserForCleanup: Browser | undefined;
    if (deviceEvidenceEnabled) this.qualificationDeviceStarts += 1;
    try {
      if (deviceEvidenceEnabled && !this.qualificationDevice) {
        this.qualificationDevice = createPipeWireQualificationDevice();
      }
      if (deviceEvidenceEnabled) {
        // A host-device qualification must use getUserMedia; fake-media launch
        // flags would prove a Chromium fixture rather than the OS device.
        const permissions = new Set(spec.permissions ?? []);
        permissions.add('microphone');
        spec = { ...spec, permissions: [...permissions] };
        await this.qualificationDevice;
      }
      if (fakeMicrophoneWav) {
        // Fake capture devices only serve pages that were granted microphone
        // access; grant it at the context level so getUserMedia never prompts.
        const permissions = new Set(spec.permissions ?? []);
        permissions.add('microphone');
        spec = { ...spec, permissions: [...permissions] };
      }
      const audioCapability = await this.browserManager.getHostAudioCapability();
      const audioStrategy: AudioStrategy = await this.browserManager.getAudioStrategy();
      const browser = await this.browserManager.getBrowser(
        deviceEvidenceEnabled ? undefined : fakeMicrophoneWav,
        audioStrategy
      );
      browserForCleanup = browser;

      // Build context (includes actualViewport with source attribution)
      const {
        context,
        storageOrigins,
        harPath,
        tracePath,
        videoDir,
        serviceWorkerController,
        recordingInitializer,
        actualViewport,
      } = await buildContext(browser, spec, this.config, audioStrategy);
      contextForCleanup = context;

      const page = await context.newPage();
      let audioDeviceEvidence: BrowserCaptureDeviceEvidence | undefined;
      if (deviceEvidenceEnabled) {
        if (!spec.base_url) {
          throw new Error(
            'device evidence requires base_url so mediaDevices can be verified on an application origin'
          );
        }
        // about:blank has an opaque origin and does not expose mediaDevices.
        // Verify on the caller's application origin in a short-lived page so
        // the session's normal initial page remains unchanged for the caller.
        const evidencePage = await context.newPage();
        try {
          await evidencePage.goto(spec.base_url, {
            waitUntil: 'domcontentloaded',
            timeout: 30_000,
          });
          const evidence = await verifyBrowserCaptureDevice(
            evidencePage,
            PIPEWIRE_QUALIFICATION_DEVICE_NAME.replaceAll(' ', '_')
          );
          if (!evidence.enumerated) {
            throw new Error(`browser did not enumerate ${PIPEWIRE_QUALIFICATION_DEVICE_NAME}`);
          }
          audioDeviceEvidence = evidence;
          logger.info('browser: host capture device evidence recorded', evidence);
          if (fakeMicrophoneWav) {
            const qualificationDevice = this.qualificationDevice
              ? await this.qualificationDevice
              : null;
            if (!qualificationDevice)
              throw new Error('host capture qualification device is unavailable');
            const startPlayback = () => qualificationDevice.startWavLoop(
              fakeMicrophoneWav,
              spec.audio_playback_pause_ms ?? 0,
              spec.audio_playback_start_delay_ms ?? 0,
              (error) => {
                audioPlaybackFailure = error.message;
                logger.error(
                  scopedLog(LogContext.RECORDING, 'host capture qualification playback failed'),
                  {
                    sessionId,
                    error: error.message,
                  }
                );
              }
            );
            audioPlaybackRestart = async () => {
              await audioPlaybackStop?.();
              const restartedPlayback = startPlayback();
              await restartedPlayback.ready;
              audioPlaybackStop = restartedPlayback.stop;
            };
            if (spec.audio_playback_defer_start) {
              logger.info('browser: host capture qualification playback deferred', {
                path: fakeMicrophoneWav,
                pauseMs: spec.audio_playback_pause_ms ?? 0,
              });
            } else {
              const initialPlayback = startPlayback();
              await initialPlayback.ready;
              audioPlaybackStop = initialPlayback.stop;
              logger.info('browser: host capture qualification playback started', {
                path: fakeMicrophoneWav,
                pauseMs: spec.audio_playback_pause_ms ?? 0,
              });
            }
          }
        } finally {
          await evidencePage.close().catch(() => undefined);
        }
      }
      if (audioStrategy === 'synthetic_sink') {
        // Init scripts do not retroactively patch the initial about:blank page.
        await applySilentSinkToCurrentPage(page, generateSilentSinkPatch());
      }

      // Log page errors (warn level - these are important signals for debugging)
      page.on('pageerror', (err: unknown) => {
        logger.warn(scopedLog(LogContext.BROWSER, 'page error'), {
          sessionId,
          error: getErrorMessage(err),
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

      const pageIdMap = new Map<string, Page>();
      const pageToIdMap = new WeakMap<Page, string>();
      const initialPageId = crypto.randomUUID();
      pageIdMap.set(initialPageId, page);
      pageToIdMap.set(page, initialPageId);

      // Create recording pipeline manager (eager instantiation)
      // This allows early verification and ensures the pipeline is ready before recording starts
      const pipelineManager = new RecordingPipelineManager(page, context, recordingInitializer, {
        sessionId,
        logger,
        getDriverPageId: (target) => pageToIdMap.get(target),
      });

      // Create session state
      const session: SessionState = {
        id: sessionId,
        ownerExecutionId: spec.execution_id,
        leaseId: uuidv4(),
        browser,
        audioCapability,
        audioStrategy,
        audioDeviceEvidence,
        // Deferred qualification playback has no handle until the first
        // turn-boundary restart. Keep a closure in the session state so the
        // later stop endpoint observes the current handle rather than the
        // undefined construction-time value.
        audioPlaybackStop: audioPlaybackRestart ? async () => { await audioPlaybackStop?.(); } : undefined,
        audioPlaybackRestart,
        audioPlaybackFailure: () => audioPlaybackFailure,
        context,
        storageOrigins,
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
        pageIdMap,
        pageToIdMap,
        activeMocks: new Map(),
        // Idempotency: Track executed instructions for replay safety
        instructionReceipts: new Map(),
        lastInstructionSequence: 0,
        // Service worker control
        serviceWorkerController,
        // Recording context initializer (binding + init script)
        recordingInitializer,
        // Recording pipeline manager (single source of truth for recording state)
        pipelineManager,
      };

      this.sessions.set(sessionId, session);
      trackSessionPageStack(session);

      try {
        // Setup diagnostic logging for redirect loop debugging
        // Enable with DIAGNOSTIC_LOGGING=true environment variable
        setupDiagnosticLogging(context, sessionId);

        // Initialize recording pipeline (early verification)
        // This runs injection and verification so the pipeline is ready before recording starts
        // The promise is stored in session.pipelineReadyPromise so consumers can await it
        const pipelineReadyPromise = pipelineManager
          .initialize()
          .then(() => {
            return pipelineManager.verifyPipeline({ timeoutMs: 5000, retries: 1 });
          })
          .then((verification) => {
            if (
              verification.scriptLoaded &&
              verification.scriptReady &&
              verification.inMainContext
            ) {
              logger.debug(scopedLog(LogContext.SESSION, 'recording pipeline verified'), {
                sessionId,
                handlersCount: verification.handlersCount,
              });
              return true;
            } else {
              logger.warn(
                scopedLog(LogContext.SESSION, 'recording pipeline verification incomplete'),
                {
                  sessionId,
                  verification,
                  hint: 'Recording may require re-verification on first use',
                }
              );
              return false;
            }
          })
          .catch((err: unknown) => {
            logger.warn(scopedLog(LogContext.SESSION, 'recording pipeline init failed'), {
              sessionId,
              error: getErrorMessage(err),
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

        // Performance tracing (Tier 0 CDP trace + web-vitals). Started here —
        // after the page exists but before the first navigate instruction — so
        // the web-vitals init script applies to the page under test and the CDP
        // trace spans the entire session. Best-effort: a failure leaves the
        // session fully functional, just without a perf artifact.
        if (spec.required_capabilities?.performance_trace) {
          const perfDir =
            spec.artifact_paths?.perf_dir?.trim() ||
            (spec.artifact_paths?.root?.trim()
              ? path.join(spec.artifact_paths.root.trim(), 'performance')
              : '');
          if (perfDir) {
            await injectWebVitalsObserver(context);
            const tracer = new PerformanceTracer(perfDir);
            await tracer.start(page);
            session.perfTracer = tracer;
          } else {
            logger.warn(
              scopedLog(LogContext.TELEMETRY, 'performance trace requested without artifact path'),
              {
                sessionId,
                hint: 'set artifact_paths.perf_dir or artifact_paths.root to capture a perf trace',
              }
            );
          }
        }

        // Accessibility snapshot. Registered here (no session-spanning state to
        // start) so the output dir + capability gate are captured at start; the
        // snapshot itself fires at session close, on the final settled page —
        // after wait_for and any interaction, the same point the final screenshot
        // fires. Best-effort: a missing artifact path just skips the capability.
        if (spec.required_capabilities?.accessibility) {
          const accessibilityDir =
            spec.artifact_paths?.accessibility_dir?.trim() ||
            (spec.artifact_paths?.root?.trim()
              ? path.join(spec.artifact_paths.root.trim(), 'accessibility')
              : '');
          if (accessibilityDir) {
            session.accessibilitySnapshotter = new AccessibilitySnapshotter(accessibilityDir);
          } else {
            logger.warn(
              scopedLog(
                LogContext.TELEMETRY,
                'accessibility snapshot requested without artifact path'
              ),
              {
                sessionId,
                hint: 'set artifact_paths.accessibility_dir or artifact_paths.root to capture an AX snapshot',
              }
            );
          }
        }

        // Session-level instrumentation hook (no-op by default).
        await safeInvoke(this.instrumentation.onSessionStart?.bind(this.instrumentation), {
          sessionId,
          executionId: spec.execution_id,
        });

        // Return actualViewport from buildContext (includes source attribution)
        return { sessionId, leaseId: session.leaseId, reused: false, createdAt, actualViewport };
      } catch (error) {
        // The map insertion precedes several async initializers. A failed
        // initializer must release browser resources and capacity immediately.
        await this.closeSession(sessionId).catch((closeError: unknown) => {
          logger.warn(scopedLog(LogContext.CLEANUP, 'partial session cleanup failed'), {
            sessionId,
            error: getErrorMessage(closeError),
          });
        });
        throw error;
      }
    } catch (error) {
      // Device qualification is prepared before the session is inserted into
      // the manager. If browser/context setup or device verification fails at
      // that boundary, the normal session cleanup path cannot see it.
      if (!this.sessions.has(sessionId)) {
        await audioPlaybackStop?.().catch(() => undefined);
        await contextForCleanup?.close().catch(() => undefined);
        await browserForCleanup?.close().catch(() => undefined);
        await this.closeQualificationDeviceIfIdle();
      }
      throw error;
    } finally {
      if (deviceEvidenceEnabled) {
        this.qualificationDeviceStarts -= 1;
        await this.closeQualificationDeviceIfIdle();
      }
    }
  }

  /** Attach the normal workflow/session machinery to an owned desktop target. */
  private async startAppTargetSessionInternal(
    spec: SessionSpec,
    target: AppTargetSpec
  ): Promise<SessionCreationResult> {
    validateAppTargetSpec(target);
    validateAppTargetCapabilities(spec.required_capabilities);
    const validationContext = spec.validation_context;
    if (!validationContext) {
      throw new Error('Electron validation context is required');
    }
    if (validationContext.context_id !== target.context_id) {
      throw new Error('Electron validation context does not match target context');
    }
    if (
      validationContext.scenario_name !== target.scenario_name ||
      validationContext.artifact_digest !== target.artifact_digest
    ) {
      throw new Error('Electron validation context does not match target identity');
    }
    // The provider workflow identity identifies the selected catalog asset,
    // while spec.workflow_id identifies BAS's internal execution/index record.
    // Adhoc executions deliberately use different values for those domains;
    // the target, scenario, artifact, context, and lease invariants above are
    // the shared validation-cell identity.
    if (
      validationContext.target_id !== target.target_id ||
      !validationContext.workflow_id?.trim()
    ) {
      throw new Error('Electron validation context does not match session identity');
    }
    if (!validationContext.isolation_lease_id?.trim()) {
      throw new Error('Electron validation context requires an isolation lease');
    }
    await verifyAppTargetRenderer(target);
    const browser = await this.browserManager.connectOverCDP(
      target.cdp_endpoint,
      target.target_kind === 'android-webview'
    );
    let sessionId = '';
    try {
      const contexts = browser.contexts();
      if (contexts.length !== 1) {
        throw new Error(
          `Electron target must expose exactly one browser context; found ${contexts.length}`
        );
      }
      const context = contexts[0];
      if (!context) throw new Error('Electron target browser context is missing');
      const extraHeaders = spec.browser_profile?.extra_headers;
      if (extraHeaders && Object.keys(extraHeaders).length > 0) {
        await context.setExtraHTTPHeaders(extraHeaders);
      }
      const page = await selectAppTargetPage(context.pages(), target);
      sessionId = uuidv4();
      const createdAt = new Date();
      const pageIdMap = new Map<string, Page>();
      const pageToIdMap = new WeakMap<Page, string>();
      const pageId = crypto.randomUUID();
      pageIdMap.set(pageId, page);
      pageToIdMap.set(page, pageId);
      const recordingInitializer = createRecordingContextInitializer({ logger });
      await recordingInitializer.initialize(context);
      const serviceWorkerController = new ServiceWorkerController(
        spec.execution_id,
        spec.service_worker_control || { mode: 'allow' }
      );
      await serviceWorkerController.enable(page);
      const pipelineManager = new RecordingPipelineManager(page, context, recordingInitializer, {
        sessionId,
        logger,
        getDriverPageId: (targetPage) => pageToIdMap.get(targetPage),
      });
      const session: SessionState = {
        id: sessionId,
        ownerExecutionId: spec.execution_id,
        leaseId: uuidv4(),
        browser,
        externalTarget: true,
        audioStrategy: 'host_device',
        context,
        storageOrigins: new Set(),
        page,
        spec,
        createdAt,
        lastUsedAt: new Date(),
        tracing: false,
        video: false,
        phase: 'ready',
        instructionCount: 0,
        frameStack: [],
        pages: [page],
        currentPageIndex: 0,
        pageIdMap,
        pageToIdMap,
        activeMocks: new Map(),
        instructionReceipts: new Map(),
        lastInstructionSequence: 0,
        serviceWorkerController,
        recordingInitializer,
        pipelineManager,
      };
      this.sessions.set(sessionId, session);
      trackSessionPageStack(session);
      setupDiagnosticLogging(context, sessionId);
      session.pipelineReadyPromise = pipelineManager
        .initialize()
        .then(() => pipelineManager.verifyPipeline({ timeoutMs: 5000, retries: 1 }))
        .then(
          (verification) =>
            verification.scriptLoaded && verification.scriptReady && verification.inMainContext
        )
        .catch((error: unknown) => {
          logger.warn(scopedLog(LogContext.SESSION, 'external target recording init failed'), {
            sessionId,
            error: getErrorMessage(error),
          });
          return false;
        });
      await safeInvoke(this.instrumentation.onSessionStart?.bind(this.instrumentation), {
        sessionId,
        executionId: spec.execution_id,
      });
      metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
      metrics.sessionCount.set({ state: 'total' }, this.sessions.size);
      const viewport = page.viewportSize() || spec.viewport;
      return {
        sessionId,
        leaseId: session.leaseId,
        reused: false,
        createdAt,
        actualViewport: {
          width: viewport.width,
          height: viewport.height,
          source: 'requested',
          reason: 'Using the controlled Electron renderer viewport',
        },
      };
    } catch (error) {
      await browser.close().catch(() => undefined);
      throw error;
    }
  }

  /**
   * Get session by ID
   */
  getSession(sessionId: string): SessionState {
    const session = this.peekSession(sessionId);
    session.lastUsedAt = new Date();
    return session;
  }

  // Observation must not extend a session lease. Health and observability use
  // this side-effect-free lookup so polling cannot defeat idle cleanup.
  peekSession(sessionId: string): SessionState {
    const session = this.sessions.get(sessionId);
    if (!session) {
      throw new SessionNotFoundError(sessionId);
    }
    return session;
  }

  /** Admit work only for the current, unreleased execution lease. */
  getSessionForLease(sessionId: string, executionId: string, leaseId: string): SessionState {
    const session = this.peekSession(sessionId);
    if (!executionId || !leaseId || session.ownerExecutionId !== executionId ||
        session.leaseId !== leaseId || session.leaseReleasedAt) {
      throw new SessionNotFoundError(sessionId);
    }
    return session;
  }

  /**
   * Releases an execution's lease without transferring ownership. Only the
   * active owner and exact lease token may release it; stale cleanup from an
   * earlier execution is harmless.
   */
  releaseExecutionLease(sessionId: string, executionId: string, leaseId: string): boolean {
    const session = this.sessions.get(sessionId);
    if (!session || session.ownerExecutionId !== executionId || session.leaseId !== leaseId) {
      return false;
    }
    session.leaseReleasedAt = new Date();
    session.lastUsedAt = session.leaseReleasedAt;
    logger.info(scopedLog(LogContext.SESSION, 'execution lease released'), {
      sessionId,
      executionId,
      leaseId,
    });
    return true;
  }

  /** Restart deterministic host playback only for its active session owner. */
  async restartAudioPlayback(sessionId: string, executionId: string, leaseId: string): Promise<boolean> {
    const session = this.sessions.get(sessionId);
    if (!session || session.ownerExecutionId !== executionId || session.leaseId !== leaseId) {
      return false;
    }
    if (!session.audioPlaybackRestart) return false;
    await session.audioPlaybackRestart();
    logger.info(scopedLog(LogContext.SESSION, 'session audio playback restarted'), { sessionId, executionId });
    return true;
  }

  /** Stop deterministic host playback at the owner-defined turn boundary. */
  async stopAudioPlayback(sessionId: string, executionId: string, leaseId: string): Promise<boolean> {
    const session = this.sessions.get(sessionId);
    if (!session || session.ownerExecutionId !== executionId || session.leaseId !== leaseId) {
      return false;
    }
    if (!session.audioPlaybackStop) return true;
    await session.audioPlaybackStop();
    logger.info(scopedLog(LogContext.SESSION, 'session audio playback stopped'), { sessionId, executionId });
    return true;
  }

  /** Close only if this exact execution still owns the active lease. */
  async closeSessionForLease(
    sessionId: string,
    executionId: string,
    leaseId: string
  ): Promise<SessionCloseResult> {
    const session = this.sessions.get(sessionId) ?? this.closingSessions.get(sessionId)?.session;
    if (!session || session.ownerExecutionId !== executionId || session.leaseId !== leaseId) {
      throw new SessionNotFoundError(sessionId);
    }
    return this.closeSession(sessionId);
  }

  // Recovery-only path; normal callers must continue to use the lease guard.
  async forceCloseSession(sessionId: string): Promise<SessionCloseResult> {
    return this.closeSession(sessionId);
  }

  /**
   * Export cookies, localStorage and IndexedDB authentication state for a session.
   */
  async getStorageState(sessionId: string): Promise<Awaited<ReturnType<BrowserContext['storageState']>>> {
    // Exporting storage is an operation on the session, not observation. Keep
    // the idle lease alive while the caller is actively using it.
    const session = this.getSession(sessionId);
    return session.context.storageState({ indexedDB: true });
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
    if (!session) return false;
    if (!session.pipelineManager || session.pipelineManager.isReady()) return true;
    if (!session.pipelineReadyPromise) return false;

    let deadline: ReturnType<typeof setTimeout> | undefined;
    try {
      return await Promise.race([
        session.pipelineReadyPromise,
        new Promise<boolean>((resolve) => { deadline = setTimeout(() => resolve(false), timeoutMs); }),
      ]);
    } catch {
      return false;
    } finally {
      if (deadline) clearTimeout(deadline);
    }
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
    return !session.instructionInFlight && canAcceptInstructions(session.phase);
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
  getSessionInfo(sessionId: string): SessionInfo {
    return inspectSession(this.peekSession(sessionId));
  }

  /**
   * Reset session (navigate to about:blank, clear state)
   */
  async resetSession(sessionId: string): Promise<void> {
    const session = this.peekSession(sessionId);
    if (session.externalTarget) {
      throw new Error('resetting an external target is refused; the target owner controls its browser state');
    }
    if (session.phase !== 'resetting' && !canTransition(session.phase, 'resetting')) {
      throw new Error(`Cannot reset session while ${session.phase}`);
    }
    const pending = this.resettingSessions.get(sessionId);
    if (pending) return pending;
    // Reserve before recording flush or browser I/O can yield. Failed attempts
    // stay resetting and may be explicitly retried; closing remains terminal.
    session.phase = 'resetting';
    session.lastUsedAt = new Date();
    const reset = (async () => {
      if (session.instructionSettlement) await session.instructionSettlement;
      await resetSessionState(session);
      if (session.phase === 'resetting') session.phase = 'ready';
    })().finally(() => { this.resettingSessions.delete(sessionId); });
    this.resettingSessions.set(sessionId, reset);
    return reset;
  }

  /**
   * Close session and cleanup resources
   *
   * Hardened to be idempotent - safe to call concurrently from multiple sources
   * (e.g., explicit close and idle cleanup).
   */
  async closeSession(sessionId: string): Promise<SessionCloseResult> {
    const pending = this.closingSessions.get(sessionId);
    if (pending) return pending.result;
    const session = this.peekSession(sessionId);
    const previousPhase = session.phase;
    session.phase = 'closing';
    session.instructionInterrupted = session.instructionInFlight === true;

    const closing = (async (): Promise<SessionCloseResult> => {
      await safeInvoke(this.instrumentation.onSessionClose?.bind(this.instrumentation), {
        sessionId, executionId: session.spec.execution_id, leaseId: session.leaseId,
      });
      const startTime = Date.now();
      logger.info(scopedLog(LogContext.SESSION, 'closing'), { sessionId, previousPhase });
      try {
        // A reset may already own browser I/O. Join its settlement before disposal;
        // close can still recover a reset that failed partway through clearing.
        await this.resettingSessions.get(sessionId)?.catch(() => undefined);
        // Teardown interrupts an admitted browser operation at the active page
        // (or detaches an external target), then joins its uncertain receipt
        // before releasing the session lease.
        const videoPaths = await teardownSessionResources(session);
        this.sessions.delete(sessionId);
        metrics.sessionDuration.observe(Date.now() - startTime);
        logger.info(scopedLog(LogContext.SESSION, 'closed'), { sessionId, cleanupDurationMs: Date.now() - startTime });
        return { videoPaths, tracePath: session.tracePath, harPath: session.harPath };
      } catch (error) {
        logger.error(scopedLog(LogContext.SESSION, 'close failed; recovery ownership retained'), {
          sessionId, error: error instanceof Error ? error.message : String(error),
        });
        throw error;
      } finally {
        metrics.sessionCount.set({ state: 'active' }, this.getActiveSessionCount());
        metrics.sessionCount.set({ state: 'total' }, this.sessions.size);
        await this.closeQualificationDeviceIfIdle();
      }
    })().finally(() => { this.closingSessions.delete(sessionId); });
    this.closingSessions.set(sessionId, { session, result: closing });
    return closing;
  }

  // Session lookup functions moved to session-decisions.ts:
  // - findByExecutionId (was findSessionByExecutionId)
  // - findByLabels (was findReusableSession)
  // - findIdleSessions (new - encapsulates idle detection)

  /**
   * Get count of active sessions
   */
  private getActiveSessionCount(): number {
    return countActiveSessions(this.sessions.values(), this.config.session.idleTimeoutMs);
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

      const failures = await this.closeSessions(idleSessions);
      metrics.sessionCount.set({ state: 'idle' }, 0);
      if (failures.length) throw new AggregateError(failures, 'Idle session cleanup incomplete');
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
  getSessionSummary(): SessionSummary {
    return summarizeSessions(this.sessions.values(), this.config);
  }

  /**
   * Get detailed list of all sessions for observability/diagnostics.
   * Returns non-sensitive session metadata.
   */
  getSessionList(): SessionListEntry[] {
    return listSessions(this.sessions.values(), this.config);
  }

  /** Attempt every selected close, keeping failed sessions owned. */
  private async closeSessions(sessionIds: string[]): Promise<unknown[]> {
    const results = await Promise.allSettled(sessionIds.map((id) => this.closeSession(id)));
    return results.flatMap((result) => result.status === 'rejected' ? [result.reason] : []);
  }

  async shutdown(): Promise<void> {
    logger.info('session-manager: shutting down', { sessionCount: this.sessions.size });
    const failures = await this.closeSessions([...this.sessions.keys()]);
    try {
      await this.browserManager.shutdown();
      await this.closeQualificationDeviceIfIdle();
    } catch (error) {
      failures.push(error);
    }
    if (failures.length) throw new AggregateError(failures, 'Session manager shutdown incomplete');
    logger.info('session-manager: shutdown complete');
  }
}
