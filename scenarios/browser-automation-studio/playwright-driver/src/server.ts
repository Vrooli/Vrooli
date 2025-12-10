import { createServer, type IncomingMessage, type ServerResponse } from 'http';
import { loadConfig } from './config';
import { SessionManager, SessionCleanup } from './session';
import { handlerRegistry } from './handlers';
import {
  NavigationHandler,
  InteractionHandler,
  WaitHandler,
  AssertionHandler,
  ExtractionHandler,
  ScreenshotHandler,
  UploadHandler,
  DownloadHandler,
  ScrollHandler,
  FrameHandler,
  SelectHandler,
  KeyboardHandler,
  CookieStorageHandler,
  GestureHandler,
  TabHandler,
  NetworkHandler,
  DeviceHandler,
} from './handlers';
import {
  createRouter,
  handleHealth,
  handleSessionStart,
  handleSessionRun,
  handleSessionReset,
  handleSessionClose,
  handleSessionStorageState,
  handleRecordStart,
  handleRecordStop,
  handleRecordStatus,
  handleRecordActions,
  handleValidateSelector,
  handleReplayPreview,
  handleRecordNavigate,
  handleRecordScreenshot,
  handleRecordInput,
  handleRecordFrame,
  handleRecordViewport,
} from './routes';
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

  // Register handlers
  registerHandlers();

  logger.info('server: handlers registered', {
    count: handlerRegistry.getHandlerCount(),
    types: handlerRegistry.getSupportedTypes(),
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
  const router = createRouter();

  // Health check
  router.get('/health', async (req, res) => {
    await handleHealth(req, res, sessionManager);
  });

  // Session management
  router.post('/session/start', async (req, res) => {
    await handleSessionStart(req, res, sessionManager, config);
  });

  router.post('/session/:id/run', async (req, res, params) => {
    await handleSessionRun(
      req,
      res,
      params.id,
      sessionManager,
      handlerRegistry,
      config,
      appLogger,
      metrics
    );
  });

  router.get('/session/:id/storage-state', async (req, res, params) => {
    await handleSessionStorageState(req, res, params.id, sessionManager);
  });

  router.post('/session/:id/reset', async (req, res, params) => {
    await handleSessionReset(req, res, params.id, sessionManager);
  });

  router.post('/session/:id/close', async (req, res, params) => {
    await handleSessionClose(req, res, params.id, sessionManager);
  });

  // Record mode
  router.post('/session/:id/record/start', async (req, res, params) => {
    await handleRecordStart(req, res, params.id, sessionManager, config);
  });

  router.post('/session/:id/record/stop', async (req, res, params) => {
    await handleRecordStop(req, res, params.id, sessionManager);
  });

  router.get('/session/:id/record/status', async (req, res, params) => {
    await handleRecordStatus(req, res, params.id, sessionManager);
  });

  router.get('/session/:id/record/actions', async (req, res, params) => {
    await handleRecordActions(req, res, params.id, sessionManager);
  });

  router.post('/session/:id/record/validate-selector', async (req, res, params) => {
    await handleValidateSelector(req, res, params.id, sessionManager, config);
  });

  router.post('/session/:id/record/replay-preview', async (req, res, params) => {
    await handleReplayPreview(req, res, params.id, sessionManager, config);
  });

  router.post('/session/:id/record/navigate', async (req, res, params) => {
    await handleRecordNavigate(req, res, params.id, sessionManager, config);
  });

  router.post('/session/:id/record/screenshot', async (req, res, params) => {
    await handleRecordScreenshot(req, res, params.id, sessionManager, config);
  });

  router.post('/session/:id/record/input', async (req, res, params) => {
    await handleRecordInput(req, res, params.id, sessionManager, config);
  });

  router.get('/session/:id/record/frame', async (req, res, params) => {
    await handleRecordFrame(req, res, params.id, sessionManager, config);
  });

  router.post('/session/:id/record/viewport', async (req, res, params) => {
    await handleRecordViewport(req, res, params.id, sessionManager, config);
  });

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

/**
 * Register all instruction handlers
 */
function registerHandlers(): void {
  // Phase 1 handlers (13 instruction types)
  handlerRegistry.register(new NavigationHandler());
  handlerRegistry.register(new InteractionHandler());
  handlerRegistry.register(new WaitHandler());
  handlerRegistry.register(new AssertionHandler());
  handlerRegistry.register(new ExtractionHandler());
  handlerRegistry.register(new ScreenshotHandler());
  handlerRegistry.register(new UploadHandler());
  handlerRegistry.register(new DownloadHandler());
  handlerRegistry.register(new ScrollHandler());

  // Phase 2 handlers (5 instruction types)
  handlerRegistry.register(new FrameHandler());
  handlerRegistry.register(new SelectHandler());
  handlerRegistry.register(new KeyboardHandler());
  handlerRegistry.register(new CookieStorageHandler());

  // Phase 3 handlers (10 instruction types)
  handlerRegistry.register(new GestureHandler());
  handlerRegistry.register(new TabHandler());
  handlerRegistry.register(new NetworkHandler());
  handlerRegistry.register(new DeviceHandler());
}

// Start server
main().catch((error) => {
  console.error('Failed to start server:', error);
  process.exit(1);
});
