import { createServer, type IncomingMessage, type ServerResponse } from 'http';
import { loadConfig, logConfigTierWarnings, getConfigSummary, type Config } from './config';
import { SessionManager, SessionCleanup } from './session';
import * as handlers from './handlers';
import * as routes from './routes';
import * as observability from './observability';
import { sendError, sendJson } from './middleware';
import { createLogger, setLogger, logger, metrics, createMetricsServer } from './utils';
import { SERVER_DRAIN_TIMEOUT_MS, SERVER_DRAIN_INTERVAL_MS } from './constants';
import { createDirectFrameServer, type DirectFrameServer } from './frame-streaming/websocket';
import { FaultController } from './fault-control';

function requireRouteParam(
  res: ServerResponse,
  params: Record<string, string>,
  key: string
): string | null {
  const value = params[key];
  if (!value) {
    sendJson(res, 400, {
      error: {
        code: 'MISSING_PARAM',
        message: `Missing route parameter: ${key}`,
      },
    });
    return null;
  }
  return value;
}

/**
 * Main Playwright Driver Server
 *
 * Entry point for the TypeScript-based Playwright driver
 */
async function main(): Promise<void> {
  // Load configuration
  const config = loadConfig();

  // Setup logger
  const appLogger = createLogger(config);
  setLogger(appLogger);

  logger.info('server: starting', {
    version: '2.0.0',
    port: config.server.port,
    host: config.server.host,
    logLevel: config.logging.level,
    metricsEnabled: config.metrics.enabled,
    configStatus: getConfigSummary(),
  });

  // Log warnings about modified configuration options (Tier 2/3)
  logConfigTierWarnings((msg, data) => {
    logger.info(msg, data);
  });

  // Create session manager
  const sessionManager = new SessionManager(config);

  // Start session cleanup task
  const cleanup = new SessionCleanup(sessionManager, config);
  cleanup.start();

  // Verify browser can launch (P0 hardening - catch Chromium issues early)
  const browserError = await sessionManager.verifyBrowserLaunch();
  if (browserError) {
    logger.error('server: browser verification failed - sessions will fail', {
      error: browserError,
      hint: 'Common causes: missing Chromium, sandbox issues, insufficient memory',
    });
    // Continue running - health endpoint will report error state
    // This allows operators to diagnose via /health without restart loops
  }

  // Register instruction handlers
  registerInstructionHandlers();

  logger.info('server: handlers registered', {
    count: handlers.handlerRegistry.getHandlerCount(),
    types: handlers.handlerRegistry.getSupportedTypes(),
  });

  // Create metrics server if enabled
  let metricsServer: ReturnType<typeof createServer> | null = null;
  if (config.metrics.enabled) {
    try {
      metricsServer = await createMetricsServer(config.metrics.port, config.server.host);
    } catch (error) {
      logger.warn('server: metrics server failed to start, continuing without metrics', {
        error: error instanceof Error ? error.message : String(error),
        port: config.metrics.port,
      });
      // Continue without metrics - this is non-fatal
    }
  }

  // Setup router with all routes
  const router = setupRoutes(sessionManager, cleanup, config, appLogger, new FaultController());

  // Create main HTTP server
  const server = createServer((req: IncomingMessage, res: ServerResponse) => {
    const pathname = new URL(req.url || '/', `http://localhost`).pathname;
    const method = req.method || 'GET';

    logger.debug('request: received', {
      method,
      path: pathname,
    });

    void router.handle(req, res).catch((error) => {
      logger.error('request: handler error', {
        method,
        path: pathname,
        error: error instanceof Error ? error.message : String(error),
      });
      sendError(res, error as Error, pathname);
    });
  });

  // Set server timeout (default Node.js timeout is 2 minutes, we need more for long-running playwright operations)
  server.timeout = config.server.requestTimeout;
  server.keepAliveTimeout = config.server.requestTimeout + 5000; // Slightly longer than request timeout
  server.headersTimeout = config.server.requestTimeout + 10000; // Slightly longer than keepAlive

  // Direct frame server: lets UI clients stream frames from the driver without
  // relaying through the API hub. Its port is allocated by the scenario, not
  // derived from the main port.
  const directFramePort = config.frameStreaming.directPort;
  const directFrameServer = createDirectFrameServer(directFramePort, config.server.host);
  directFrameServer.start();

  // Make direct frame server available globally for frame manager
  (global as { directFrameServer?: DirectFrameServer }).directFrameServer = directFrameServer;

  // Start listening
  server.listen(config.server.port, config.server.host, () => {
    logger.info('server: listening', {
      port: config.server.port,
      host: config.server.host,
      url: `http://${config.server.host}:${config.server.port}`,
      requestTimeout: config.server.requestTimeout,
    });

    // Emit explicit "ready" signal for operators/orchestrators
    // This is the key signal that the driver is operational and accepting traffic
    logger.info('server: ready', {
      status: browserError ? 'degraded' : 'ok',
      healthEndpoint: `http://${config.server.host}:${config.server.port}/health`,
      metricsEndpoint: config.metrics.enabled
        ? `http://${config.server.host}:${config.metrics.port}/metrics`
        : 'disabled',
      directFrameEndpoint: `ws://${config.server.host}:${directFramePort}/frames`,
      browserVerified: !browserError,
    });
  });

  // Track active requests for graceful shutdown
  let activeRequests = 0;
  let isShuttingDown = false;

  // Wrap request handler to track active requests
  const originalEmit = server.emit.bind(server);
  server.emit = function (event: string, ...args: unknown[]) {
    if (event === 'request') {
      activeRequests++;
      const res = args[1] as ServerResponse;
      res.on('finish', () => {
        activeRequests--;
      });
      res.on('close', () => {
        // Handle aborted requests
        if (!res.writableEnded) {
          activeRequests--;
        }
      });
    }
    return originalEmit(event, ...args);
  } as typeof server.emit;

  // Graceful shutdown with request draining.
  //
  // exitCode distinguishes an operator-requested stop from a fault. Exiting 0
  // on a fault makes the supervisor report "exited normally", which hid a
  // driver death mid-suite behind a clean-looking restart.
  const shutdown = async (signal: string, exitCode = 0): Promise<void> => {
    if (isShuttingDown) {
      logger.warn('server: shutdown already in progress, ignoring signal', { signal });
      return;
    }
    isShuttingDown = true;

    logger.info('server: shutdown initiated', { signal, activeRequests });

    // Stop accepting new connections immediately
    server.close(() => {
      logger.info('server: http closed (no longer accepting connections)');
    });

    // Stop cleanup task and wait for any in-flight cleanup to complete
    await cleanup.stop();

    // Close metrics server
    if (metricsServer) {
      metricsServer.close(() => {
        logger.info('server: metrics closed');
      });
    }

    // Close direct frame server
    directFrameServer.stop();

    // Wait for in-flight requests to complete (with timeout)
    // Timeouts from constants.ts
    const drainStart = Date.now();

    while (activeRequests > 0 && Date.now() - drainStart < SERVER_DRAIN_TIMEOUT_MS) {
      logger.debug('server: draining active requests', {
        remaining: activeRequests,
        elapsedMs: Date.now() - drainStart,
      });
      await new Promise((resolve) => setTimeout(resolve, SERVER_DRAIN_INTERVAL_MS));
    }

    if (activeRequests > 0) {
      logger.warn('server: drain timeout, proceeding with shutdown', {
        remainingRequests: activeRequests,
        drainTimeoutMs: SERVER_DRAIN_TIMEOUT_MS,
      });
    } else {
      logger.info('server: all requests drained');
    }

    // Shutdown session manager (close all browser sessions)
    await sessionManager.shutdown();

    logger.info('server: shutdown complete', { exitCode });
    process.exit(exitCode);
  };

  process.on('SIGTERM', () => {
    void shutdown('SIGTERM');
  });
  process.on('SIGINT', () => {
    void shutdown('SIGINT');
  });

  // An uncaught exception can leave module state inconsistent, so the process
  // still goes down — but with a non-zero code so the supervisor and its logs
  // name it a fault rather than a normal exit.
  process.on('uncaughtException', (error) => {
    logger.error('server: uncaught exception', {
      error: error.message,
      stack: error.stack,
    });
    void shutdown('uncaughtException', 1);
  });

  // A stray rejection is a bug on one code path, not a reason to destroy every
  // live browser session. Tearing the driver down here turned a single orphaned
  // page.screenshot promise into a suite-wide outage: every in-flight execution
  // failed with connection-refused while the exit looked clean. Log it loudly,
  // keep serving, and let the failing path surface on its own terms.
  process.on('unhandledRejection', (reason) => {
    logger.error('server: unhandled rejection (continuing)', {
      reason: reason instanceof Error ? reason.message : String(reason),
      stack: reason instanceof Error ? reason.stack : undefined,
    });
  });
}

// =============================================================================
// Route & Handler Registration
// =============================================================================

/**
 * All instruction handlers, instantiated once.
 * Handlers are stateless - they receive context per-execution.
 */
const INSTRUCTION_HANDLERS = [
  // Core browser automation
  new handlers.NavigationHandler(),
  new handlers.InteractionHandler(),
  new handlers.WaitHandler(),
  new handlers.AssertionHandler(),
  new handlers.ExtractionHandler(),
  new handlers.ScreenshotHandler(),
  new handlers.ScrollHandler(),
  // File I/O
  new handlers.UploadHandler(),
  new handlers.DownloadHandler(),
  // Advanced interaction
  new handlers.FrameHandler(),
  new handlers.SelectHandler(),
  new handlers.KeyboardHandler(),
  new handlers.CookieStorageHandler(),
  new handlers.ServiceWorkerHandler(),
  new handlers.GestureHandler(),
  new handlers.TabHandler(),
  // Network & device
  new handlers.NetworkHandler(),
  new handlers.DeviceHandler(),
];

/**
 * Register all instruction handlers with the global registry.
 */
function registerInstructionHandlers(): void {
  for (const handler of INSTRUCTION_HANDLERS) {
    handlers.handlerRegistry.register(handler);
  }
}

/**
 * Setup all HTTP routes.
 *
 * Route organization:
 * - /health: Health check
 * - /session/*: Session lifecycle (start, run, reset, close)
 * - /session/:id/record/*: Record mode (start, stop, status, actions, validation)
 */
function setupRoutes(
  sessionManager: SessionManager,
  sessionCleanup: SessionCleanup,
  config: Config,
  appLogger: typeof logger,
  faultController: FaultController
): routes.Router {
  const router = routes.createRouter();

  // Health check
  router.get('/health', (req, res) => {
    routes.handleHealth(req, res, sessionManager);
    return Promise.resolve();
  });

  // Observability (unified health, monitoring, diagnostics)
  const observabilityDeps: observability.ObservabilityRouteDependencies = {
    sessionManager,
    sessionCleanup,
    config,
  };
  router.get('/observability', async (req, res) => {
    await observability.handleObservability(req, res, observabilityDeps);
  });
  router.post('/observability/refresh', (req, res) => {
    observability.handleObservabilityRefresh(req, res);
    return Promise.resolve();
  });
  router.post('/observability/diagnostics/run', (req, res) => {
    observability.handleDiagnosticsRun(req, res, observabilityDeps);
    return Promise.resolve();
  });
  router.get('/observability/metrics', async (req, res) => {
    await observability.handleMetrics(req, res, observabilityDeps);
  });
  router.get('/observability/sessions', (req, res) => {
    observability.handleSessionList(req, res, observabilityDeps);
    return Promise.resolve();
  });
  router.post('/observability/cleanup/run', async (req, res) => {
    await observability.handleCleanupRun(req, res, observabilityDeps);
  });
  // Runtime configuration management
  router.get('/observability/config/runtime', (req, res) => {
    observability.handleConfigRuntime(req, res);
    return Promise.resolve();
  });
  router.put('/observability/config/:env_var', (req, res, params) => {
    const envVar = requireRouteParam(res, params, 'env_var');
    if (!envVar) {
      return Promise.resolve();
    }
    observability.handleConfigUpdate(req, res, envVar);
    return Promise.resolve();
  });
  router.delete('/observability/config/:env_var', (req, res, params) => {
    const envVar = requireRouteParam(res, params, 'env_var');
    if (!envVar) {
      return Promise.resolve();
    }
    observability.handleConfigReset(req, res, envVar);
    return Promise.resolve();
  });
  // Autonomous pipeline test (creates temp session if needed)
  router.post('/observability/pipeline-test', (req, res) => {
    observability.handlePipelineTest(req, res, observabilityDeps);
    return Promise.resolve();
  });
  router.post('/test-control/faults/arm', async (req, res) => routes.handleFaultArm(req, res, config, faultController));
  router.get('/test-control/faults', (req, res) => { routes.handleFaultSnapshot(req, res, config, faultController); return Promise.resolve(); });
  router.post('/test-control/faults/disarm', async (req, res) => routes.handleFaultDisarm(req, res, config, faultController));

  router.get('/artifacts', async (req, res) => {
    await routes.handleArtifactDownload(req, res);
  });

  // Session lifecycle
  router.post('/session/start', async (req, res) => {
    await routes.handleSessionStart(req, res, sessionManager, config, faultController);
  });
  router.post('/session/:id/run', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionRun(req, res, sessionId, sessionManager, handlers.handlerRegistry, config, appLogger, metrics);
  });
  router.get('/session/:id/storage-state', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionStorageState(req, res, sessionId, sessionManager);
  });

  // Service worker management
  router.get('/session/:id/service-workers', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionServiceWorkers(req, res, sessionId, sessionManager);
  });
  router.delete('/session/:id/service-workers', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionServiceWorkersDelete(req, res, sessionId, sessionManager);
  });
  router.delete('/session/:id/service-workers/:scopeURL', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    const scopeUrl = requireRouteParam(res, params, 'scopeURL');
    if (!scopeUrl) {
      return;
    }
    await routes.handleSessionServiceWorkerDelete(req, res, sessionId, scopeUrl, sessionManager);
  });

  router.post('/session/:id/reset', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionReset(req, res, sessionId, sessionManager);
  });
  router.post('/session/:id/release', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionRelease(req, res, sessionId, sessionManager);
  });
  router.post('/session/:id/close', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionClose(req, res, sessionId, sessionManager);
  });
  router.post('/session/:id/force-close', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionForceClose(req, res, sessionId, sessionManager, config);
  });

  // Record mode lifecycle
  router.post('/session/:id/record/start', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordStart(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/stop', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordStop(req, res, sessionId, sessionManager);
  });
  router.get('/session/:id/record/status', (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return Promise.resolve();
    }
    routes.handleRecordStatus(req, res, sessionId, sessionManager);
    return Promise.resolve();
  });
  router.get('/session/:id/record/actions', (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return Promise.resolve();
    }
    routes.handleRecordActions(req, res, sessionId, sessionManager);
    return Promise.resolve();
  });
  router.get('/session/:id/record/debug', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordDebug(req, res, sessionId, sessionManager);
  });
  router.post('/session/:id/record/pipeline-test', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordPipelineTest(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/external-url-test', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordExternalUrlTest(req, res, sessionId, sessionManager, config);
  });

  // Record mode validation & interaction
  router.post('/session/:id/record/validate-selector', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleValidateSelector(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/replay-preview', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleReplayPreview(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/navigate', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordNavigate(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/reload', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordReload(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/go-back', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordGoBack(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/go-forward', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordGoForward(req, res, sessionId, sessionManager, config);
  });
  router.get('/session/:id/record/navigation-state', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordNavigationState(req, res, sessionId, sessionManager, config);
  });
  router.get('/session/:id/record/navigation-stack', (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return Promise.resolve();
    }
    routes.handleRecordNavigationStack(req, res, sessionId, sessionManager, config);
    return Promise.resolve();
  });
  router.post('/session/:id/record/screenshot', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordScreenshot(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/input', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordInput(req, res, sessionId, sessionManager, config);
  });
  router.get('/session/:id/record/frame', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordFrame(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/viewport', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordViewport(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/stream-settings', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleStreamSettings(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/new-page', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordNewPage(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/record/active-page', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleRecordActivePage(req, res, sessionId, sessionManager, config);
  });

  // AI Navigation
  router.post('/session/:id/ai-navigate', async (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return;
    }
    await routes.handleSessionAINavigate(req, res, sessionId, sessionManager, config);
  });
  router.post('/session/:id/ai-navigate/abort', (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return Promise.resolve();
    }
    routes.handleSessionAINavigateAbort(req, res, sessionId, sessionManager, config);
    return Promise.resolve();
  });
  router.post('/session/:id/ai-navigate/resume', (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return Promise.resolve();
    }
    routes.handleSessionAINavigateResume(req, res, sessionId, sessionManager, config);
    return Promise.resolve();
  });
  router.get('/session/:id/ai-navigate/status', (req, res, params) => {
    const sessionId = requireRouteParam(res, params, 'id');
    if (!sessionId) {
      return Promise.resolve();
    }
    routes.handleSessionAINavigateStatus(req, res, sessionId, sessionManager, config);
    return Promise.resolve();
  });
  router.get('/ai/models', (req, res) => {
    routes.handleListAIModels(req, res);
    return Promise.resolve();
  });

  return router;
}

// Start server
main().catch((error) => {
  console.error('Failed to start server:', error);
  process.exit(1);
});
