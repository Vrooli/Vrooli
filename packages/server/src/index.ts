import { i18nConfig } from "@local/shared";
import { ApolloServer } from "apollo-server-express";
import cookie from "cookie";
import cors from "cors";
import express from "express";
import fs from "fs";
import { graphqlUploadExpress } from "graphql-upload";
import i18next from "i18next";
import { app } from "./app";
import * as auth from "./auth/request";
import { schema } from "./endpoints";
import * as restRoutes from "./endpoints/rest";
import { logger } from "./events/logger";
import { io } from "./io";
import { context, depthLimit } from "./middleware";
import { initializeRedis } from "./redisConn";
import { initCountsCronJobs, initEventsCronJobs, initExpirePremiumCronJob, initGenerateEmbeddingsCronJob, initModerationCronJobs, initSitemapCronJob, initStatsCronJobs } from "./schedules";
import { server, SERVER_URL } from "./server";
import { setupStripe, setupValyxa } from "./services";
import { chatSocketHandlers, notificationSocketHandlers } from "./sockets";
import { loadSecrets } from "./utils/loadSecrets";
import { setupDatabase } from "./utils/setupDatabase";

const debug = process.env.NODE_ENV === "development";

const main = async () => {
    logger.info("Starting server...");

    // Load .env variables from secrets location
    loadSecrets(process.env.NODE_ENV as "development" | "production");

    // Check for required .env variables
    const requiredEnvs = ["PROJECT_DIR", "VITE_SERVER_LOCATION", "LETSENCRYPT_EMAIL", "VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY"];
    for (const env of requiredEnvs) {
        if (!process.env[env]) {
            logger.error(`🚨 ${env} not in environment variables. Stopping server`, { trace: "0007" });
            process.exit(1);
        }
    }

    // Check for JWT public/private key files
    const requiredKeyFiles = ["jwt_priv.pem", "jwt_pub.pem"];
    for (const keyFile of requiredKeyFiles) {
        try {
            const key = fs.readFileSync(`${process.env.PROJECT_DIR}/${keyFile}`);
            if (!key) {
                logger.error(`🚨 ${keyFile} not found. Stopping server`, { trace: "0448" });
                process.exit(1);
            }
        } catch (error) {
            logger.error(`🚨 ${keyFile} not found. Stopping server`, { trace: "0449", error });
            process.exit(1);
        }
    }

    // Initialize translations
    await i18next.init(i18nConfig(debug));

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
    app.use((req, res, next) => {
        req.cookies = cookie.parse(req.headers.cookie || "");
        next();
    });

    // Set up health check endpoint
    app.get("/healthcheck", (_req, res) => {
        res.status(200).send("OK");
    });

    // For authentication
    app.use(auth.authenticate);

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
        if (req.originalUrl === "/webhook/stripe") {
            next();
        } else {
            express.json({ limit: "20mb" })(req, res, next);
        }
    });

    // Set up external services (including webhooks)
    setupStripe(app);
    setupValyxa(app);

    // Set static folders
    // app.use(`/api/images`, express.static(`${process.env.PROJECT_DIR}/data/images`));

    // Set up REST API
    Object.keys(restRoutes).forEach((key) => {
        app.use("/api/v2/rest", restRoutes[key]);
    });

    // Set up image uploading for GraphQL
    app.use("/api/v2/graphql", graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }));

    // Apollo server for latest API version
    const apollo_options_latest = new ApolloServer({
        cache: "bounded",
        introspection: debug,
        schema,
        context: (c) => context(c), // Allows request and response to be included in the context
        validationRules: [depthLimit(13)], // Prevents DoS attack from arbitrarily-nested query
    });
    // Start server
    await apollo_options_latest.start();
    // Configure server with ExpressJS settings and path
    apollo_options_latest.applyMiddleware({
        app,
        path: "/api/v2/graphql",
        cors: false,
    });
    // Additional Apollo server that wraps the latest version. Uses previous endpoint, and transforms the schema
    // to be compatible with the latest version.
    // const apollo_options_previous = new ApolloServer({
    //     cache: 'bounded' as any,
    //     introspection: debug,
    //     schema: schema,
    //     context: (c) => context(c), // Allows request and response to be included in the context
    //     validationRules: [depthLimit(10)] // Prevents DoS attack from arbitrarily-nested query
    // });
    // // Start server
    // await apollo_options_previous.start();
    // // Configure server with ExpressJS settings and path
    // apollo_options_previous.applyMiddleware({
    //     app,
    //     path: `/api/v2`,
    //     cors: false
    // });

    // Set up websocket server
    // Pass the session to the socket, after it's been authenticated
    io.use(auth.authenticateSocket);
    // Listen for new WebSocket connections
    io.on("connection", (socket) => {
        console.log("a user connected");
        // Add handlers
        chatSocketHandlers(io, socket);
        notificationSocketHandlers(io, socket);
        // Handle disconnect
        socket.on("disconnect", () => {
            console.log("user disconnected");
        });
    });

    // Start Express server
    server.listen(5329);

    // Start cron jobs
    initStatsCronJobs();
    initEventsCronJobs();
    initCountsCronJobs();
    initSitemapCronJob();
    initModerationCronJobs();
    initExpirePremiumCronJob();
    initGenerateEmbeddingsCronJob();

    logger.info(`🚀 Server running at ${SERVER_URL}`);
};

main();
