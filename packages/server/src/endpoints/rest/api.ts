import { api_create, api_findMany, api_findOne, api_update } from "../generated";
import { ApiEndpoints } from "../logic/api";
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
});
