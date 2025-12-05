import { createServer, type IncomingMessage, type ServerResponse } from 'http';
import { parse } from 'url';
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
  handleHealth,
  handleSessionStart,
  handleSessionRun,
  handleSessionReset,
  handleSessionClose,
  handleRecordStart,
  handleRecordStop,
  handleRecordStatus,
  handleRecordActions,
  handleValidateSelector,
  handleReplayPreview,
} from './routes';
import { send404, send405, sendError } from './middleware';
import { createLogger, setLogger, logger, metrics } from './utils';

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

  logger.info('Starting Playwright Driver v2.0', {
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
    logger.error('⚠️  Browser launch verification failed - sessions will fail', {
      error: browserError,
      hint: 'Common causes: missing Chromium, sandbox issues, insufficient memory',
    });
    // Continue running - health endpoint will report error state
    // This allows operators to diagnose via /health without restart loops
  }

  // Register handlers
  registerHandlers();

  logger.info('Registered handlers', {
    count: handlerRegistry.getHandlerCount(),
    types: handlerRegistry.getSupportedTypes(),
  });

  // Create metrics server if enabled
  let metricsServer: ReturnType<typeof createServer> | null = null;
  if (config.metrics.enabled) {
    try {
      metricsServer = await createMetricsServer(config.metrics.port);
    } catch (error) {
      logger.warn('Failed to start metrics server, continuing without metrics', {
        error: error instanceof Error ? error.message : String(error),
        port: config.metrics.port,
      });
      // Continue without metrics - this is non-fatal
    }
  }

  // Create main HTTP server
  const server = createServer(async (req: IncomingMessage, res: ServerResponse) => {
    const parsedUrl = parse(req.url || '', true);
    const pathname = parsedUrl.pathname || '/';
    const method = req.method || 'GET';

    logger.debug('Request received', {
      method,
      path: pathname,
    });

    try {
      // Route handling
      if (pathname === '/health' && method === 'GET') {
        await handleHealth(req, res, sessionManager);
      } else if (pathname === '/session/start' && method === 'POST') {
        await handleSessionStart(req, res, sessionManager, config);
      } else if (pathname.match(/^\/session\/[^/]+\/run$/) && method === 'POST') {
        const sessionId = pathname.split('/')[2];
        await handleSessionRun(
          req,
          res,
          sessionId,
          sessionManager,
          handlerRegistry,
          config,
          appLogger,
          metrics
        );
      } else if (pathname.match(/^\/session\/[^/]+\/reset$/) && method === 'POST') {
        const sessionId = pathname.split('/')[2];
        await handleSessionReset(req, res, sessionId, sessionManager);
      } else if (pathname.match(/^\/session\/[^/]+\/close$/) && method === 'POST') {
        const sessionId = pathname.split('/')[2];
        await handleSessionClose(req, res, sessionId, sessionManager);
      } else if (pathname.match(/^\/session\/[^/]+\/record\/start$/) && method === 'POST') {
        const sessionId = pathname.split('/')[2];
        await handleRecordStart(req, res, sessionId, sessionManager, config);
      } else if (pathname.match(/^\/session\/[^/]+\/record\/stop$/) && method === 'POST') {
        const sessionId = pathname.split('/')[2];
        await handleRecordStop(req, res, sessionId, sessionManager);
      } else if (pathname.match(/^\/session\/[^/]+\/record\/status$/) && method === 'GET') {
        const sessionId = pathname.split('/')[2];
        await handleRecordStatus(req, res, sessionId, sessionManager);
      } else if (pathname.match(/^\/session\/[^/]+\/record\/actions$/) && method === 'GET') {
        const sessionId = pathname.split('/')[2];
        await handleRecordActions(req, res, sessionId, sessionManager);
      } else if (pathname.match(/^\/session\/[^/]+\/record\/validate-selector$/) && method === 'POST') {
        const sessionId = pathname.split('/')[2];
        await handleValidateSelector(req, res, sessionId, sessionManager, config);
      } else if (pathname.match(/^\/session\/[^/]+\/record\/replay-preview$/) && method === 'POST') {
        const sessionId = pathname.split('/')[2];
        await handleReplayPreview(req, res, sessionId, sessionManager, config);
      } else if (pathname === '/health' && method !== 'GET') {
        send405(res, ['GET']);
      } else if (pathname === '/session/start' && method !== 'POST') {
        send405(res, ['POST']);
      } else {
        send404(res, `Path not found: ${pathname}`);
      }
    } catch (error) {
      logger.error('Request handler error', {
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
    logger.info('Server listening', {
      port: config.server.port,
      host: config.server.host,
      url: `http://${config.server.host}:${config.server.port}`,
      requestTimeout: config.server.requestTimeout,
    });
  });

  // Graceful shutdown
  const shutdown = async (signal: string) => {
    logger.info('Received shutdown signal', { signal });

    // Stop accepting new connections
    server.close(() => {
      logger.info('HTTP server closed');
    });

    // Stop cleanup task
    cleanup.stop();

    // Close metrics server
    if (metricsServer) {
      metricsServer.close(() => {
        logger.info('Metrics server closed');
      });
    }

    // Shutdown session manager
    await sessionManager.shutdown();

    logger.info('Shutdown complete');
    process.exit(0);
  };

  process.on('SIGTERM', () => shutdown('SIGTERM'));
  process.on('SIGINT', () => shutdown('SIGINT'));

  // Handle uncaught errors
  process.on('uncaughtException', (error) => {
    logger.error('Uncaught exception', {
      error: error.message,
      stack: error.stack,
    });
    shutdown('uncaughtException');
  });

  process.on('unhandledRejection', (reason) => {
    logger.error('Unhandled rejection', {
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

/**
 * Create metrics server
 */
function createMetricsServer(port: number): Promise<ReturnType<typeof createServer>> {
  return new Promise((resolve, reject) => {
    const server = createServer(async (req, res) => {
      if (req.url === '/metrics') {
        res.setHeader('Content-Type', metrics.getRegistry().contentType);
        const metricsOutput = await metrics.getMetrics();
        res.end(metricsOutput);
      } else {
        res.statusCode = 404;
        res.end('Not found');
      }
    });

    server.on('error', (error: NodeJS.ErrnoException) => {
      if (error.code === 'EADDRINUSE') {
        reject(new Error(`Metrics port ${port} is already in use`));
      } else {
        reject(error);
      }
    });

    server.listen(port, '0.0.0.0', () => {
      logger.info('Metrics server listening', {
        port,
        url: `http://0.0.0.0:${port}/metrics`,
      });
      resolve(server);
    });
  });
}

// Start server
main().catch((error) => {
  console.error('Failed to start server:', error);
  process.exit(1);
});
