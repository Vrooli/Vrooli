import { ApolloServer } from "@apollo/server";
import { expressMiddleware } from "@apollo/server/express4";
import { ApolloServerPluginDrainHttpServer } from "@apollo/server/plugin/drainHttpServer";
import { HttpStatus, i18nConfig } from "@local/shared";
import cookie from "cookie";
import cors from "cors";
import express from "express";
import { graphqlUploadExpress } from "graphql-upload";
import i18next from "i18next";
import path from "path";
import { fileURLToPath } from "url";
import { app } from "./app";
import * as auth from "./auth/request";
import { schema } from "./endpoints";
import * as restRoutes from "./endpoints/rest";
import { logger } from "./events/logger";
import { context, depthLimit } from "./middleware";
import { ModelMap } from "./models/base";
import { initializeRedis } from "./redisConn";
import { SERVER_URL, server } from "./server";
import { setupStripe } from "./services";
import { io } from "./sockets/io";
import { chatSocketRoomHandlers } from "./sockets/rooms/chat";
import { runSocketRoomHandlers } from "./sockets/rooms/run";
import { userSocketRoomHandlers } from "./sockets/rooms/user";
import { setupEmailQueue } from "./tasks/email/queue";
import { setupExportQueue } from "./tasks/export/queue";
import { setupImportQueue } from "./tasks/import/queue";
import { setupLlmQueue } from "./tasks/llm/queue";
import { LlmServiceRegistry } from "./tasks/llm/registry";
import { setupLlmTaskQueue } from "./tasks/llmTask/queue";
import { setupPushQueue } from "./tasks/push/queue";
import { setupRunQueue } from "./tasks/run/queue";
import { setupSandboxQueue } from "./tasks/sandbox";
import { setupSmsQueue } from "./tasks/sms/queue";
import { initializeProfanity } from "./utils/censor";
import { setupDatabase } from "./utils/setupDatabase";

const debug = process.env.NODE_ENV === "development";
const QUERY_DEPTH_LIMIT = 13;

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export async function initSingletons() {
    // Initialize translations
    await i18next.init(i18nConfig(debug));

    // Initialize singletons
    await ModelMap.init();
    await LlmServiceRegistry.init();

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

    // Set up health check endpoint
    app.get("/healthcheck", (_req, res) => {
        res.status(HttpStatus.Ok).send("OK");
    });

    // For authentication
    app.use((req, res, next) => {
        // Exclude webhooks, as they have their own authentication methods
        if (req.originalUrl.startsWith("/webhooks")) {
            next();
        } else {
            auth.authenticate(req, res, next);
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
    Object.keys(restRoutes).forEach((key) => {
        app.use("/api/v2/rest", restRoutes[key]);
    });

    // Set up image uploading for GraphQL
    app.use("/api/v2/graphql", graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }));

    // GraphQL server for latest API version, if needed. We use GraphQL to generate 
    // types, so this needs to run at least in development.
    if (debug) {
        const apolloServerLatest = new ApolloServer({
            cache: "bounded",
            introspection: debug,
            schema,
            plugins: [ApolloServerPluginDrainHttpServer({ httpServer: server })],
            validationRules: [depthLimit(QUERY_DEPTH_LIMIT)], // Prevents DoS attack from arbitrarily-nested query
        });
        // Start server
        await apolloServerLatest.start();
        // Configure server with ExpressJS settings and path
        app.use(
            "/api/v2/graphql",
            expressMiddleware(apolloServerLatest, {
                context: async ({ req, res }) => context({ req, res }), // Pass req and res to your context function
            }),
        );
    }

    // Set up websocket server
    // Pass the session to the socket, after it's been authenticated
    io.use(auth.authenticateSocket);
    // Listen for new WebSocket connections
    io.on("connection", (socket) => {
        // Add handlers
        chatSocketRoomHandlers(socket);
        runSocketRoomHandlers(socket);
        userSocketRoomHandlers(socket);
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
export * from "./builders";
export * from "./db/instance";
export * from "./events";
export * from "./models";
export * from "./notify";
export * from "./redisConn";
export * from "./server";
export * from "./sockets/events";
export * from "./tasks";
export * from "./utils";
export * from "./validators";

