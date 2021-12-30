import express from 'express';
import cookieParser from 'cookie-parser';
import * as auth from './auth/auth';
import cors from "cors";
import { ApolloServer } from 'apollo-server-express';
import { depthLimit } from './depthLimit';
import { graphqlUploadExpress } from 'graphql-upload';
import { API_VERSION, SERVER_URL } from '@local/shared';
import { schema } from './schema';
import { context } from './context';
import { envVariableExists } from './utils/envVariableExists';
import { setupDatabase } from './utils/setupDatabase';

const main = async () => {
    console.info('Starting server...')

    // Check for required .env variables
    if (['JWT_SECRET', 'REACT_APP_SERVER_ROUTE'].some(name => !envVariableExists(name))) process.exit(1);

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
        `http://app.${process.env.REACT_APP_SITE_NAME}`,
        `http://www.app.${process.env.REACT_APP_SITE_NAME}`,
        `https://${process.env.REACT_APP_SITE_NAME}`,
        `https://www.app.${process.env.REACT_APP_SITE_NAME}`
    ]
    app.use(cors({
        credentials: true,
        origin: origins
    }))

    // Set static folders
    app.use(`${process.env.REACT_APP_SERVER_ROUTE}/images`, express.static(`${process.env.PROJECT_DIR}/data/uploads`));

    // Set up image uploading
    app.use(`${process.env.REACT_APP_SERVER_ROUTE}/${API_VERSION}`, graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }),)

    // Set up GraphQL using Apollo
    // Context trickery allows request and response to be included in the context
    const apollo_options = new ApolloServer({
        introspection: process.env.NODE_ENV === 'development',
        schema: schema,
        context: (c) => context(c),
        validationRules: [depthLimit(8)] // Prevents DoS attack from arbitrarily-nested query
    });
    await apollo_options.start();
    apollo_options.applyMiddleware({
        app,
        path: `${process.env.REACT_APP_SERVER_ROUTE}/${API_VERSION}`,
        cors: false
    });
    // Start Express server
    app.listen(process.env.VIRTUAL_PORT);

    console.info(`🚀 Server running at ${SERVER_URL}`)
}

main();