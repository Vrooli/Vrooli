import { initIdGenerator } from "@vrooli/shared";
import cookie from "cookie";
import cors from "cors";
import express from "express";
import path from "path";
import { fileURLToPath } from "url";
import { app } from "./app.js";
import { AuthService } from "./auth/auth.js";
import { DbProvider } from "./db/provider.js";
import { initRestApi } from "./endpoints/rest.js";
import { logger } from "./events/logger.js";
import { CacheService, getRedisUrl } from "./redisConn.js";
import { SERVER_PORT, SERVER_URL, server } from "./server.js";
import { BillingWorker } from "./services/billing.js";
import { setupErrorReporting } from "./services/errorReporting.js";
import { setupHealthCheck } from "./services/health.js";
import { setupMCP } from "./services/mcp/index.js";
import { setupMetrics } from "./services/metrics.js";
import { ResourceRegistry } from "./services/resources/ResourceRegistry.js";
import { setupStripe } from "./services/stripe.js";
import { SocketService } from "./sockets/io.js";
import { chatSocketRoomHandlers } from "./sockets/rooms/chat.js";
import { runSocketRoomHandlers } from "./sockets/rooms/run.js";
import { userSocketRoomHandlers } from "./sockets/rooms/user.js";
import { QueueService } from "./tasks/queues.js";
import { checkImageProcessingCapabilities } from "./utils/fileStorage.js";
import { initSingletons } from "./utils/singletons.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function main() {

    // Check for required .env variables
    const requiredEnvs = [
        "JWT_PRIV", // Private key for JWT tokens
        "JWT_PUB", // Public key for JWT tokens
        "PROJECT_DIR", // Path to the project directory
        "VITE_SERVER_LOCATION", // Location of the server
        "VAPID_PUBLIC_KEY", // Public key for VAPID
        "VAPID_PRIVATE_KEY", // Private key for VAPID
        "WORKER_ID", // Worker ID (e.g. pod ordinal) for Snowflake IDs
    ];
    for (const env of requiredEnvs) {
        if (!process.env[env]) {
            logger.error(`🚨 ${env} not in environment variables. Stopping server`, { trace: "0007" });
            process.exit(1);
        }
    }

    // Set snowflake worker ID
    initIdGenerator(parseInt(process.env.WORKER_ID ?? "0"));

    // Initialize singletons
    await initSingletons();

    // Initialize services that depend on external connections

    // 1. Setup Database
    try {
        await DbProvider.init(); // This includes seeding
    } catch (dbError) {
        logger.error("🚨 Critical: Database initialization or seeding failed. Server might not function correctly.", { error: dbError });
        // Depending on the severity, you might want to process.exit(1) here
        // For now, allowing to continue to see if other services can start, or if retries handle it.
    }

    // 2. Initialize Resource Registry
    try {
        const resourceRegistry = ResourceRegistry.getInstance();
        await resourceRegistry.initialize();
        logger.info("✅ Resource Registry initialized successfully");
        
        // Update AI services with ResourceRegistry for enhanced health checking
        const { AIServiceRegistry } = await import("./services/response/registry.js");
        AIServiceRegistry.get().updateServicesWithResourceRegistry(resourceRegistry);
        logger.info("✅ AI services updated with ResourceRegistry");
    } catch (resourceError) {
        logger.error("⚠️ Resource Registry initialization failed. Local resources won't be available.", { error: resourceError });
        // Non-critical - server can run without local resources
    }

    // 3. Initialize Redis Connection for CacheService
    try {
        CacheService.get(); // This will call its constructor and internal `ensure`
    } catch (cacheError) {
        const err = cacheError as Error;
        logger.error("🚨 Critical: CacheService initial get() or ensure() failed.", {
            errorName: err?.name,
            errorMessage: err?.message,
            errorStack: err?.stack,
            errorObj: JSON.stringify(cacheError, Object.getOwnPropertyNames(cacheError)), // Detailed logging
        });
    }

    // 4. Initialize Redis Connection for QueueService
    const queueService = QueueService.get();
    try {
        const redisUrl = getRedisUrl();
        await queueService.init(redisUrl);
        await queueService.initializeAllQueues();

    } catch (queueServiceError) {
        const err = queueServiceError as Error;
        logger.error("🚨 Critical: QueueService initialization, Redis connection, or task queue setup failed.", {
            errorName: err?.name,
            errorMessage: err?.message,
            errorStack: err?.stack,
            errorObj: JSON.stringify(queueServiceError, Object.getOwnPropertyNames(queueServiceError)), // Detailed logging
        });
    }

    // 5. Start event bus and its workers
    try {
        await BillingWorker.start();
    } catch (billingError) {
        logger.error("🚨 Critical: BillingWorker failed to start.", { error: billingError });
    }

    // 6. Check image processing capabilities (non-critical)
    try {
        await checkImageProcessingCapabilities();
    } catch (error) {
        logger.error("Failed to check image processing capabilities", {
            error: error instanceof Error ? error.message : String(error),
            message: "Image processing features may be limited",
        });
    }

    // // For parsing application/xwww-
    // app.use(express.urlencoded({ extended: false }));

    // For parsing cookies
    app.use((req, _res, next) => {
        req.cookies = cookie.parse(req.headers.cookie || "");
        next();
    });

    // For authentication
    app.use((req, res, next) => {
        // Exclude healthcheck endpoints and webhooks from authentication
        // - Healthcheck needs to be accessible for monitoring without auth
        // - Webhooks have their own authentication methods
        if (req.originalUrl.startsWith("/webhooks") || req.originalUrl.startsWith("/healthcheck")) {
            next();
        } else {
            // Include everything else, which should include REST endpoints and websockets
            AuthService.authenticateRequest(req, res, next);
        }
    });

    // Set up health check endpoint
    setupHealthCheck(app);

    // Set up metrics endpoint
    setupMetrics(app);

    // Set up error reporting endpoint
    setupErrorReporting(app);

    // Cross-Origin access. Accepts requests from localhost and dns
    // Disable if API is open to the public
    // API is open, so allow all origins
    app.use(cors({
        credentials: true,
        origin: true, //safeOrigins(),
    }));

    // For parsing application/json. 
    app.use((req: express.Request, res: express.Response, next: express.NextFunction) => {
        // Exclude on stripe webhook endpoint
        if (req.originalUrl === "/webhooks/stripe") {
            next();
        } else {
            express.json({ limit: "20mb" })(req, res, next);
        }
    });

    // Set up external services (including webhooks)
    setupStripe(app);
    setupMCP(app);

    // Set static folders
    app.use("/ai/configs", express.static(path.join(__dirname, "../../shared/dist/ai/configs")));

    // Set up REST API
    await initRestApi(app);

    // Set up websocket server
    const socketService = SocketService.get(); // Get the initialized instance
    const ioInstance = socketService.io; // Access the io instance from the service

    // Authenticate new connections (this is not called for each event)
    ioInstance.use(AuthService.authenticateSocket);
    // Set up socket connection logic
    ioInstance.on("connection", (socket) => {
        // Add socket to maps using the service method
        socketService.addSocket(socket);

        // Handle disconnection using the service method
        socket.on("disconnect", () => {
            socketService.removeSocket(socket);
        });

        // Add handlers for each room
        chatSocketRoomHandlers(socket);
        runSocketRoomHandlers(socket);
        userSocketRoomHandlers(socket);
        // Add more room handlers as needed
    });

    // Unhandled Rejection Handler. This is a last resort for catching errors that were not caught by the application. 
    // If you see this error, please try to find its source and catch it there.
    process.on("unhandledRejection", (reason, promise) => {
        logger.error("🚨 Unhandled Rejection", { trace: "0003", reason, promise });
    });

    // Handle server startup errors
    server.on("error", (error: NodeJS.ErrnoException) => {
        if (error.code === "EADDRINUSE") {
            logger.error(`Port ${SERVER_PORT} is already in use`, {
                trace: "0008",
                port: SERVER_PORT,
                suggestions: [
                    `Kill the process using: lsof -ti :${SERVER_PORT} | xargs kill -9`,
                    `Or set a different port: PORT_API=${SERVER_PORT + 1} pnpm run develop`,
                    "Or use Docker: pnpm run develop --target docker",
                ],
            });
            process.exit(1);
        } else if (error.code === "EACCES") {
            logger.error(`Permission denied to bind to port ${SERVER_PORT}`, {
                trace: "0009",
                port: SERVER_PORT,
                suggestions: [
                    "Use a port number above 1024",
                    "Or run with elevated permissions (not recommended)",
                ],
            });
            process.exit(1);
        } else {
            logger.error("Server startup error", {
                trace: "0010",
                error: error.message,
                code: error.code,
                stack: error.stack,
            });
            process.exit(1);
        }
    });

    // Start Express server
    server.listen(SERVER_PORT);
    logger.info(`🚀 Server running at ${SERVER_URL}`);
}

// Only call this from the "server" package when not testing
if (
    process.env.NODE_ENV !== "test" &&
    (process.env.npm_package_name === "@vrooli/server" || !process.env.npm_package_name)
) {
    main();
}

// Export files for "jobs" package
export * from "./auth/bcryptWrapper.js";
export * from "./builders/index.js";
export * from "./db/provider.js";
export * from "./events/index.js";
export * from "./models/index.js";
export * from "./notify/index.js";
export * from "./redisConn.js";
export * from "./server.js";
export * from "./services/index.js";
export * from "./sockets/io.js";
export * from "./tasks/index.js";
export * from "./utils/index.js";
export * from "./validators/index.js";

// Export endpoints for integration testing
export * from "./endpoints/index.js";
export * from "./endpoints/logic/index.js";

// Export test utilities for integration testing

