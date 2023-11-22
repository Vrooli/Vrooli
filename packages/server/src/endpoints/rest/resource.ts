import { resource_create, resource_findMany, resource_findOne, resource_update } from "../generated";
import { ResourceEndpoints } from "../logic/resource";
import { setupRoutes } from "./base";

export const ResourceRest = setupRoutes({
    "/resource/:id": {
        get: [ResourceEndpoints.Query.resource, resource_findOne],
        put: [ResourceEndpoints.Mutation.resourceUpdate, resource_update],
    },
    "/resources": {
        get: [ResourceEndpoints.Query.resources, resource_findMany],
    },
    "/resource": {
        post: [ResourceEndpoints.Mutation.resourceCreate, resource_create],
    },
});
