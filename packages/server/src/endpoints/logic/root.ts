import { ApiEndpoint } from "../../types";

export type EndpointsRoot = {
    Query: {
        _empty: ApiEndpoint<Record<string, never>, string>;
    },
    Mutation: {
        _empty: ApiEndpoint<Record<string, never>, string>;
    }
}

export const RootEndpoints: EndpointsRoot = {
    Query: {
        _empty: async () => {
            return "Hello World!";
        },
    },
    Mutation: {
        _empty: async () => {
            return "Hello World!";
        },
    },
};
