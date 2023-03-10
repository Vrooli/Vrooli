import express from 'express';
import cookieParser from 'cookie-parser';
import * as auth from './auth/request';
import cors from "cors";
import { ApolloServer } from 'apollo-server-express';
import { context, depthLimit } from './middleware';
import { graphqlUploadExpress } from 'graphql-upload';
import { schema } from './endpoints';
import { setupDatabase } from './utils/setupDatabase';
import { logger } from './events/logger';
import { initializeRedis } from './redisConn';
import { i18nConfig } from '@shared/translations';
import i18next from 'i18next';
import { initStatsCronJobs } from './schedules/stats';
import { initEventsCronJobs } from './schedules/events';
import { initCountsCronJobs } from './schedules/counts';
import { initSitemapCronJob } from './schedules';

const debug = process.env.NODE_ENV === 'development';

const SERVER_URL = process.env.VITE_SERVER_LOCATION === 'local' ?
    `http://localhost:5329/api` :
    Boolean(process.env.SERVER_URL) ?
        process.env.SERVER_URL :
        `http://${process.env.SITE_IP}:5329/api`;

const main = async () => {
    logger.info('Starting server...');

    // Check for required .env variables
    const requiredEnvs = ['VITE_SERVER_LOCATION', 'JWT_SECRET', 'LETSENCRYPT_EMAIL', 'VAPID_PUBLIC_KEY', 'VAPID_PRIVATE_KEY'];
    for (const env of requiredEnvs) {
        if (!process.env[env]) {
            console.error('uh oh', process.env[env]);
            logger.error(`🚨 ${env} not in environment variables. Stopping server`, { trace: '0007' });
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
        logger.info('✅ Connected to Redis');
    } catch (error) {
        logger.error('🚨 Failed to connect to Redis', { trace: '0207', error });
    }

    const app = express();

    // // For parsing application/json
    // app.use(express.json());
    // // For parsing application/xwww-
    // app.use(express.urlencoded({ extended: false }));
    // For parsing cookies
    app.use(cookieParser(process.env.JWT_SECRET));

    // For authentication
    app.use(auth.authenticate);

    // Cross-Origin access. Accepts requests from localhost and dns
    // Disable if API is open to the public
    // API is open, so allow all origins
    app.use(cors({
        credentials: true,
        origin: true, //safeOrigins(),
    }))

    // Set static folders
    // app.use(`/api/images`, express.static(`${process.env.PROJECT_DIR}/data/images`));

    // Set up image uploading
    app.use(`/api/v2`, graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }))

    // Apollo server for latest API version
    const apollo_options_latest = new ApolloServer({
        cache: 'bounded' as any,
        introspection: debug,
        schema: schema,
        context: (c) => context(c), // Allows request and response to be included in the context
        validationRules: [depthLimit(11)] // Prevents DoS attack from arbitrarily-nested query
    });
    // Start server
    await apollo_options_latest.start();
    // Configure server with ExpressJS settings and path
    apollo_options_latest.applyMiddleware({
        app,
        path: `/api/v2`,
        cors: false
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
    // Start Express server
    app.listen(5329);

    // Start cron jobs
    initStatsCronJobs();
    initEventsCronJobs();
    initCountsCronJobs();
    initSitemapCronJob();

    logger.info( `🚀 Server running at ${SERVER_URL}`);
}

main();