import express from 'express';
import cookieParser from 'cookie-parser';
import * as auth from './auth/auth';
import cors from "cors";
import { ApolloServer } from 'apollo-server-express';
import { depthLimit } from './depthLimit';
import { graphqlUploadExpress } from 'graphql-upload';
import { schema } from './schema';
import { context } from './context';
import { setupDatabase } from './utils/setupDatabase';
import { initStatsCronJobs } from './statsLog';
import mongoose from 'mongoose';
import { genErrorCode, logger, LogLevel } from './logger';
import { initializeRedis } from './redisConn';

const SERVER_URL = process.env.REACT_APP_SERVER_LOCATION === 'local' ?
    `http://localhost:5329/api` :
    Boolean(process.env.SERVER_URL) ?
        process.env.SERVER_URL :
        `http://${process.env.SITE_IP}:5329/api`;

const main = async () => {
    logger.log(LogLevel.info, 'Starting server...');

    // Check for required .env variables
    const requiredEnvs = ['REACT_APP_SERVER_LOCATION', 'JWT_SECRET'];
    for (const env of requiredEnvs) {
        if (!process.env[env]) {
            console.error('uh oh', process.env[env]);
            logger.log(LogLevel.error, `🚨 ${env} not in environment variables. Stopping server`, { code: genErrorCode('0007') });
            process.exit(1);
        }
    }

    // Setup databases
    // Prisma
    await setupDatabase();
    // MongoDB
    try {
        mongoose.set('debug', process.env.NODE_ENV === 'development');
        await mongoose.connect(process.env.MONGO_CONN ?? '', {
            useNewUrlParser: true,
            useUnifiedTopology: true,
        } as mongoose.ConnectOptions);
        logger.log(LogLevel.info, '✅ Connected to MongoDB');
    } catch (error) {
        logger.log(LogLevel.error, '🚨 Failed to connect to MongoDB', { code: genErrorCode('0191'), error });
    }
    // Redis 
    try {
        await initializeRedis();
        logger.log(LogLevel.info, '✅ Connected to Redis');
    } catch (error) {
        logger.log(LogLevel.error, '🚨 Failed to connect to Redis', { code: genErrorCode('0207'), error });
    }

    const app = express();

    // // For parsing application/json
    // app.use(express.json());
    // // For parsing application/xwww-
    // app.use(express.urlencoded({ extended: false }));
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
    app.use(`/api/v1`, graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }))

    /**
     * Apollo Server for GraphQL
     */
    const apollo_options = new ApolloServer({
        cache: 'bounded' as any,
        introspection: process.env.NODE_ENV === 'development',
        schema: schema,
        context: (c) => context(c), // Allows request and response to be included in the context
        validationRules: [depthLimit(10)] // Prevents DoS attack from arbitrarily-nested query
    });
    // Start server
    await apollo_options.start();
    // Configure server with ExpressJS settings and path
    apollo_options.applyMiddleware({
        app,
        path: `/api/v1`,
        cors: false
    });
    // Start Express server
    app.listen(5329);

    // Start cron jobs for calculating site statistics
    initStatsCronJobs();

    logger.log(LogLevel.info, `🚀 Server running at ${SERVER_URL}`);
    console.log('yeet');
}

main();