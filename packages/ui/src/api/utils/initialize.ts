import { ApolloClient, ApolloLink, InMemoryCache, NormalizedCacheObject } from "@apollo/client";
import { onError } from "@apollo/client/link/error";
import { createUploadLink } from "apollo-upload-client";
import { useMemo } from "react";
import { removeTypename } from "./removeTypename";

let apolloClient: ApolloClient<NormalizedCacheObject>;

const createApolloClient = (): ApolloClient<NormalizedCacheObject> => {
    // Define link for error handling
    const errorLink = onError(({ graphQLErrors, networkError }) => {
        // Only developers should see these error messages
        if (process.env.NODE_ENV === "production") return;
        if (graphQLErrors) {
            graphQLErrors.forEach(({ message, locations, path }) => {
                console.error("GraphQL error occurred");
                console.error(`Path: ${path}`);
                console.error(`Location: ${locations}`);
                console.error(`Message: ${message}`);
            });
        }
        if (networkError) {
            console.error("GraphQL network error occurred", networkError);
        }
    });
    // Determine origin of API server
    let uri: string;
    // If running locally
    if (window.location.host.includes("localhost") || window.location.host.includes("192.168.0.")) {
        uri = `http://${window.location.hostname}:${process.env.VITE_PORT_SERVER ?? "5329"}/api/v2`;
    }
    // If running on server
    else {
        uri = process.env.VITE_SERVER_URL && process.env.VITE_SERVER_URL.length > 0 ?
            `${process.env.VITE_SERVER_URL}/v2` :
            `http://${process.env.VITE_SITE_IP}:${process.env.VITE_PORT_SERVER ?? "5329"}/api/v2`;
    }
    // Define link for handling file uploads
    const uploadLink = createUploadLink({
        uri,
        credentials: "include",
    });
    // Define link for removing '__typename'. This field cannot be in queries or mutations, 
    // and is sometimes tedious to remove manually
    const cleanTypenameLink = new ApolloLink((operation, forward) => {
        if (operation.variables && !operation.variables.file && !operation.variables.files) {
            operation.variables = removeTypename(operation.variables);
        }
        return forward(operation);
    });
    // Create Apollo client
    return new ApolloClient({
        cache: new InMemoryCache(),
        link: ApolloLink.from([errorLink, cleanTypenameLink, uploadLink]),
    });
};

export const initializeApollo = (): ApolloClient<NormalizedCacheObject> => {
    const _apolloClient = apolloClient ?? createApolloClient();
    if (!apolloClient) apolloClient = _apolloClient;

    return _apolloClient;
};

export const useApollo = (): ApolloClient<NormalizedCacheObject> => {
    const store = useMemo(() => initializeApollo(), []);
    return store;
};
