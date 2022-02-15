import express from 'express';
import cookieParser from 'cookie-parser';
import * as auth from './auth/auth';
import cors from "cors";
import { ApolloServer } from 'apollo-server-express';
import { depthLimit } from './depthLimit';
import { graphqlUploadExpress } from 'graphql-upload';
import { schema } from './schema';
import { context } from './context';
import { envVariableExists } from './utils/envVariableExists';
import { setupDatabase } from './utils/setupDatabase';

const SERVER_URL = process.env.REACT_APP_SERVER_LOCATION === 'local' ? 
`http://localhost:5329/api` : 
`https://app.vrooli.com/api`;

const main = async () => {
    console.info('Starting server...')

    // Check for required .env variables
    if (['JWT_SECRET'].some(name => !envVariableExists(name))) process.exit(1);

    // Setup database
    await setupDatabase();

    const app = express();

    // // For parsing application/json
    // app.use(express.json());
    // // For parsing application/xwww-
    // app.use(express.urlencoded({ extended: false }));
    app.use(cookieParser(process.env.JWT_SECRET));

    // For authentication
    app.use(auth.authenticate);

    // Cross-Origin access. Accepts requests from localhost and dns
    // If you want a public server, this can be set to ['*']
    let origins = [
        /^http:\/\/localhost(?::[0-9]+)?$/,
        'https://studio.apollographql.com',
        `http://app.vrooli.com`,
        `http://www.app.vrooli.com`,
        `https://app.vrooli.com`,
        `https://www.app.vrooli.com`
    ]
    app.use(cors({
        credentials: true,
        origin: origins
    }))

    // Set static folders
    app.use(`/api/images`, express.static(`${process.env.PROJECT_DIR}/data/uploads`));

    // Set up image uploading
    app.use(`/api/v1`, graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }),)

    /**
     * Apollo Server for GraphQL
     */
    const apollo_options = new ApolloServer({
        introspection: process.env.NODE_ENV === 'development',
        schema: schema,
        context: (c) => context(c), // Allows request and response to be included in the context
        validationRules: [depthLimit(8)] // Prevents DoS attack from arbitrarily-nested query
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

    console.info(`🚀 Server running at ${SERVER_URL}`)
}

main();