import { HttpStatus, i18nConfig } from "@local/shared";
import cookie from "cookie";
import cors from "cors";
import express from "express";
import i18next from "i18next";
import path from "path";
import { fileURLToPath } from "url";
import { app } from "./app.js";
import { AuthService } from "./auth/auth.js";
import { SessionService } from "./auth/session.js";
import { DbProvider } from "./db/provider.js";
import { initRestApi } from "./endpoints/rest.js";
import { logger } from "./events/logger.js";
import { ModelMap } from "./models/base/index.js";
import { initializeRedis } from "./redisConn.js";
import { SERVER_URL, server } from "./server.js";
import { setupStripe } from "./services/stripe.js";
import { io, sessionSockets, userSockets } from "./sockets/io.js";
import { chatSocketRoomHandlers } from "./sockets/rooms/chat.js";
import { runSocketRoomHandlers } from "./sockets/rooms/run.js";
import { userSocketRoomHandlers } from "./sockets/rooms/user.js";
import { setupEmailQueue } from "./tasks/email/queue.js";
import { setupExportQueue } from "./tasks/export/queue.js";
import { setupImportQueue } from "./tasks/import/queue.js";
import { setupLlmQueue } from "./tasks/llm/queue.js";
import { LlmServiceRegistry } from "./tasks/llm/registry.js";
import { TokenEstimationRegistry } from "./tasks/llm/tokenEstimator.js";
import { setupLlmTaskQueue } from "./tasks/llmTask/queue.js";
import { setupPushQueue } from "./tasks/push/queue.js";
import { setupRunQueue } from "./tasks/run/queue.js";
import { setupSandboxQueue } from "./tasks/sandbox/queue.js";
import { setupSmsQueue } from "./tasks/sms/queue.js";
import { initializeProfanity } from "./utils/censor.js";

const debug = process.env.NODE_ENV === "development";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export async function initSingletons() {
    // Initialize translations
    await i18next.init(i18nConfig(debug));

    // Initialize singletons
    await ModelMap.init();
    await LlmServiceRegistry.init();
    await TokenEstimationRegistry.init();

    // Initialize censor dictionary
    initializeProfanity();
}

async function main() {
    logger.info("Starting server...");

    // Check for required .env variables
    const requiredEnvs = ["JWT_PRIV", "JWT_PUB", "PROJECT_DIR", "VITE_SERVER_LOCATION", "LETSENCRYPT_EMAIL", "VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY"];
    for (const env of requiredEnvs) {
        if (!process.env[env]) {
            logger.error(`🚨 ${env} not in environment variables. Stopping server`, { trace: "0007" });
            process.exit(1);
        }
    }

    // Initialize singletons
    await initSingletons();

    // Setup queues
    await setupLlmTaskQueue();
    await setupEmailQueue();
    await setupExportQueue();
    await setupImportQueue();
    await setupLlmQueue();
    await setupPushQueue();
    await setupSandboxQueue();
    await setupSmsQueue();
    await setupRunQueue();

    // Setup databases
    // Prisma
    await setupDatabase();
    // Redis 
    try {
        await initializeRedis();
        logger.info("✅ Connected to Redis");
    } catch (error) {
        logger.error("🚨 Failed to connect to Redis", { trace: "0207", error });
    }

    // // For parsing application/xwww-
    // app.use(express.urlencoded({ extended: false }));

    // For parsing cookies
    app.use((req, _res, next) => {
        req.cookies = cookie.parse(req.headers.cookie || "");
        next();
    });

    // Set up health check endpoint before authentication
    app.get("/healthcheck", (_req, res) => {
        res.status(HttpStatus.Ok).send("OK");
    });

    // For authentication
    app.use((req, res, next) => {
        console.log("in here. Current time:", new Date());
        // Exclude webhooks, as they have their own authentication methods
        if (req.originalUrl.startsWith("/webhooks")) {
            next();
        } else {
            // Include everything else, which should include REST endpoints and websockets
            AuthService.authenticateRequest(req, res, next);
        }
    });

    // Cross-Origin access. Accepts requests from localhost and dns
    // Disable if API is open to the public
    // API is open, so allow all origins
    app.use(cors({
        credentials: true,
        origin: true, //safeOrigins(),
    }));

    // For parsing application/json. 
    app.use((req, res, next) => {
        // Exclude on stripe webhook endpoint
        if (req.originalUrl === "/webhooks/stripe") {
            next();
        } else {
            express.json({ limit: "20mb" })(req, res, next);
        }
    });

    // Set up external services (including webhooks)
    setupStripe(app);

    // Set static folders
    app.use("/ai/configs", express.static(path.join(__dirname, "../../shared/dist/ai/configs")));

    // Set up REST API
    await initRestApi(app);

    // Set up websocket server
    // Authenticate new connections (this is not called for each event)
    io.use(AuthService.authenticateSocket);
    // Set up socket connection logic
    io.on("connection", (socket) => {
        // Keep track of sockets for each user and session, so that they can be 
        // disconnected when the user logs out or revokes open sessions
        const user = SessionService.getUser(socket.session);
        const userId = user?.id;
        const sessionId = user?.session.id;
        if (userId && sessionId) {
            if (!userSockets.has(userId)) {
                userSockets.set(userId, new Set());
            }
            userSockets.get(userId)?.add(socket.id);

            if (!sessionSockets.has(sessionId)) {
                sessionSockets.set(sessionId, new Set());
            }
            sessionSockets.get(sessionId)?.add(socket.id);

            socket.on("disconnect", () => {
                userSockets.get(userId)?.delete(socket.id);
                sessionSockets.get(sessionId)?.delete(socket.id);
            });
        }

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
    server.listen(5329);

    logger.info(`🚀 Server running at ${SERVER_URL}`);
}

// Only call this from the "server" package
if (process.env.npm_package_name === "@local/server") {
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
export * from "./sockets/events.js";
export * from "./tasks/index.js";
export * from "./utils/index.js";
export * from "./validators/index.js";

