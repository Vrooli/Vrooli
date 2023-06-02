import { api_create, api_findMany, api_findOne, api_update } from "@local/shared";
import { ApiEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ApiRest = setupRoutes({
    "/api/:id": {
        get: [ApiEndpoints.Query.api, api_findOne],
        put: [ApiEndpoints.Mutation.apiUpdate, api_update],
    },
    "/apis": {
        get: [ApiEndpoints.Query.apis, api_findMany],
    },
    "/api": {
        post: [ApiEndpoints.Mutation.apiCreate, api_create],
    },
} as const);
