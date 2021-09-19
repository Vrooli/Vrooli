import { useMemo } from 'react';
import { SERVER_QUERY_URL } from '@local/shared';
import {
    ApolloClient,
    ApolloLink,
    InMemoryCache,
} from '@apollo/client';
import { createUploadLink } from 'apollo-upload-client';
import { removeTypename } from './removeTypename';

let apolloClient;

function createApolloClient() {
    // Define link for handling file uploads
    const uploadLink = createUploadLink({
        uri: SERVER_QUERY_URL,
        credentials: 'include'
    });
    // Define link for removing '__typename'. This field cannot be in queries or mutations, 
    // and is sometimes tedious to remove manually
    const cleanTypenameLink = new ApolloLink((operation, forward) => {
        if (operation.variables && !operation.variables.file && !operation.variables.files) {
          operation.variables = removeTypename(operation.variables);
        }
        return forward(operation);
      })
    // Create Apollo client
    return new ApolloClient({
        cache: new InMemoryCache(),
        link: ApolloLink.from([ cleanTypenameLink, uploadLink ]),
        onError: ({ networkError, graphQLErrors }) => {
            // Only developers should see these error messages
            if (process.env.NODE_ENV === 'production') return;
            if (graphQLErrors) {
                graphQLErrors.forEach(({ message, location, path }) => {
                    console.error('GraphQL error occurred');
                    console.error(`Path: ${path}`);
                    console.error(`Location: ${location}`);
                    console.error(`Message: ${message}`);
                })
            }
            if (networkError) {
                console.error('GraphQL network error occurred', networkError);
            }
        }
    })
}

export function initializeApollo() {
    const _apolloClient = apolloClient ?? createApolloClient();
    if (!apolloClient) apolloClient = _apolloClient;

    return _apolloClient;
}

export function useApollo() {
    const store = useMemo(() => initializeApollo(), []);
    return store;
}