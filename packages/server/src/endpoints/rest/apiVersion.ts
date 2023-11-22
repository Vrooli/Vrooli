import { apiVersion_create, apiVersion_findMany, apiVersion_findOne, apiVersion_update } from "../generated";
import { ApiVersionEndpoints } from "../logic/apiVersion";
import { setupRoutes } from "./base";

export const ApiVersionRest = setupRoutes({
    "/apiVersion/:id": {
        get: [ApiVersionEndpoints.Query.apiVersion, apiVersion_findOne],
        put: [ApiVersionEndpoints.Mutation.apiVersionUpdate, apiVersion_update],
    },
    "/apiVersions": {
        get: [ApiVersionEndpoints.Query.apiVersions, apiVersion_findMany],
    },
    "/apiVersion": {
        post: [ApiVersionEndpoints.Mutation.apiVersionCreate, apiVersion_create],
    },
});
