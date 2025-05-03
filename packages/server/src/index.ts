import { setWorkerId } from "@local/shared";
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
import { initializeRedis } from "./redisConn.js";
import { SERVER_PORT, SERVER_URL, server } from "./server.js";
import { setupHealthCheck } from "./services/health.js";
import { setupMCP } from "./services/mcp/index.js";
import { setupStripe } from "./services/stripe.js";
import { SocketService } from "./sockets/io.js";
import { chatSocketRoomHandlers } from "./sockets/rooms/chat.js";
import { runSocketRoomHandlers } from "./sockets/rooms/run.js";
import { userSocketRoomHandlers } from "./sockets/rooms/user.js";
import { setupTaskQueues } from "./tasks/setup.js";
import { initSingletons } from "./utils/singletons.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function main() {
    logger.info("Starting server...");

    // Check for required .env variables
    const requiredEnvs = [
        "JWT_PRIV", // Private key for JWT tokens
        "JWT_PUB", // Public key for JWT tokens
        "PROJECT_DIR", // Path to the project directory
        "VITE_SERVER_LOCATION", // Location of the server
        "LETSENCRYPT_EMAIL", // Email for Let's Encrypt
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
    setWorkerId(parseInt(process.env.WORKER_ID ?? "0"));

    // Initialize singletons
    await initSingletons();
    // Setup queues
    await setupTaskQueues();
    // Setup databases
    await initializeRedis();
    await DbProvider.init();

    // // For parsing application/xwww-
    // app.use(express.urlencoded({ extended: false }));

    // For parsing cookies
    app.use((req, _res, next) => {
        req.cookies = cookie.parse(req.headers.cookie || "");
        next();
    });

    // For authentication
    app.use((req, res, next) => {
        // Exclude webhooks, as they have their own authentication methods
        if (req.originalUrl.startsWith("/webhooks")) {
            next();
        } else {
            // Include everything else, which should include REST endpoints and websockets
            AuthService.authenticateRequest(req, res, next);
        }
    });

    // Set up health check endpoint
    setupHealthCheck(app);

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

    // Start Express server
    server.listen(SERVER_PORT);
    logger.info(`🚀 Server running at ${SERVER_URL}`);
}

// Only call this from the "server" package when not testing
if (
    process.env.npm_package_name === "@local/server" &&
    process.env.NODE_ENV !== "test"
) {
    main();
}

// Export files for "jobs" package
export * from "./builders/index.js";
export * from "./db/provider.js";
export * from "./events/index.js";
export * from "./models/index.js";
export * from "./notify/index.js";
export * from "./redisConn.js";
export * from "./server.js";
export * from "./sockets/io.js";
export * from "./tasks/index.js";
export * from "./utils/index.js";
export * from "./validators/index.js";

