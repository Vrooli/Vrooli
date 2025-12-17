import { createServer, type IncomingMessage, type ServerResponse } from 'http';
import { loadConfig, logConfigTierWarnings, getConfigSummary, type Config } from './config';
import { SessionManager, SessionCleanup } from './session';
import * as handlers from './handlers';
import * as routes from './routes';
import { sendError } from './middleware';
import { createLogger, setLogger, logger, metrics, createMetricsServer } from './utils';

/**
 * Main Playwright Driver Server
 *
 * Entry point for the TypeScript-based Playwright driver
 */
async function main() {
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
      metricsServer = await createMetricsServer(config.metrics.port);
    } catch (error) {
      logger.warn('server: metrics server failed to start, continuing without metrics', {
        error: error instanceof Error ? error.message : String(error),
        port: config.metrics.port,
      });
      // Continue without metrics - this is non-fatal
    }
  }

  // Setup router with all routes
  const router = setupRoutes(sessionManager, config, appLogger);

  // Create main HTTP server
  const server = createServer(async (req: IncomingMessage, res: ServerResponse) => {
    const pathname = new URL(req.url || '/', `http://localhost`).pathname;
    const method = req.method || 'GET';

    logger.debug('request: received', {
      method,
      path: pathname,
    });

    try {
      await router.handle(req, res);
    } catch (error) {
      logger.error('request: handler error', {
        method,
        path: pathname,
        error: error instanceof Error ? error.message : String(error),
      });
      sendError(res, error as Error, pathname);
    }
  });

  // Set server timeout (default Node.js timeout is 2 minutes, we need more for long-running playwright operations)
  server.timeout = config.server.requestTimeout;
  server.keepAliveTimeout = config.server.requestTimeout + 5000; // Slightly longer than request timeout
  server.headersTimeout = config.server.requestTimeout + 10000; // Slightly longer than keepAlive

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

  // Graceful shutdown with request draining
  const shutdown = async (signal: string) => {
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

    // Wait for in-flight requests to complete (with timeout)
    const drainTimeout = 30_000; // 30 seconds max drain time
    const drainStart = Date.now();
    const drainInterval = 100; // Check every 100ms

    while (activeRequests > 0 && Date.now() - drainStart < drainTimeout) {
      logger.debug('server: draining active requests', {
        remaining: activeRequests,
        elapsedMs: Date.now() - drainStart,
      });
      await new Promise((resolve) => setTimeout(resolve, drainInterval));
    }

    if (activeRequests > 0) {
      logger.warn('server: drain timeout, proceeding with shutdown', {
        remainingRequests: activeRequests,
        drainTimeoutMs: drainTimeout,
      });
    } else {
      logger.info('server: all requests drained');
    }

    // Shutdown session manager (close all browser sessions)
    await sessionManager.shutdown();

    logger.info('server: shutdown complete');
    process.exit(0);
  };

  process.on('SIGTERM', () => shutdown('SIGTERM'));
  process.on('SIGINT', () => shutdown('SIGINT'));

  // Handle uncaught errors
  process.on('uncaughtException', (error) => {
    logger.error('server: uncaught exception', {
      error: error.message,
      stack: error.stack,
    });
    shutdown('uncaughtException');
  });

  process.on('unhandledRejection', (reason) => {
    logger.error('server: unhandled rejection', {
      reason: String(reason),
    });
    shutdown('unhandledRejection');
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
  config: Config,
  appLogger: typeof logger
): routes.Router {
  const router = routes.createRouter();

  // Health check
  router.get('/health', async (req, res) => {
    await routes.handleHealth(req, res, sessionManager);
  });

  // Session lifecycle
  router.post('/session/start', async (req, res) => {
    await routes.handleSessionStart(req, res, sessionManager, config);
  });
  router.post('/session/:id/run', async (req, res, params) => {
    await routes.handleSessionRun(req, res, params.id, sessionManager, handlers.handlerRegistry, config, appLogger, metrics);
  });
  router.get('/session/:id/storage-state', async (req, res, params) => {
    await routes.handleSessionStorageState(req, res, params.id, sessionManager);
  });
  router.post('/session/:id/reset', async (req, res, params) => {
    await routes.handleSessionReset(req, res, params.id, sessionManager);
  });
  router.post('/session/:id/close', async (req, res, params) => {
    await routes.handleSessionClose(req, res, params.id, sessionManager);
  });

  // Record mode lifecycle
  router.post('/session/:id/record/start', async (req, res, params) => {
    await routes.handleRecordStart(req, res, params.id, sessionManager, config);
  });
  router.post('/session/:id/record/stop', async (req, res, params) => {
    await routes.handleRecordStop(req, res, params.id, sessionManager);
  });
  router.get('/session/:id/record/status', async (req, res, params) => {
    await routes.handleRecordStatus(req, res, params.id, sessionManager);
  });
  router.get('/session/:id/record/actions', async (req, res, params) => {
    await routes.handleRecordActions(req, res, params.id, sessionManager);
  });

  // Record mode validation & interaction
  router.post('/session/:id/record/validate-selector', async (req, res, params) => {
    await routes.handleValidateSelector(req, res, params.id, sessionManager, config);
  });
  router.post('/session/:id/record/replay-preview', async (req, res, params) => {
    await routes.handleReplayPreview(req, res, params.id, sessionManager, config);
  });
  router.post('/session/:id/record/navigate', async (req, res, params) => {
    await routes.handleRecordNavigate(req, res, params.id, sessionManager, config);
  });
  router.post('/session/:id/record/screenshot', async (req, res, params) => {
    await routes.handleRecordScreenshot(req, res, params.id, sessionManager, config);
  });
  router.post('/session/:id/record/input', async (req, res, params) => {
    await routes.handleRecordInput(req, res, params.id, sessionManager, config);
  });
  router.get('/session/:id/record/frame', async (req, res, params) => {
    await routes.handleRecordFrame(req, res, params.id, sessionManager, config);
  });
  router.post('/session/:id/record/viewport', async (req, res, params) => {
    await routes.handleRecordViewport(req, res, params.id, sessionManager, config);
  });
  router.post('/session/:id/record/stream-settings', async (req, res, params) => {
    await routes.handleStreamSettings(req, res, params.id, sessionManager, config);
  });

  return router;
}

// Start server
main().catch((error) => {
  console.error('Failed to start server:', error);
  process.exit(1);
});
