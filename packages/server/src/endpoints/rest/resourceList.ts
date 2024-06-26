import { resourceList_create, resourceList_findMany, resourceList_findOne, resourceList_update } from "../generated";
import { ResourceListEndpoints } from "../logic/resourceList";
import { setupRoutes } from "./base";

export const ResourceListRest = setupRoutes({
    "/resourceList/:id": {
        get: [ResourceListEndpoints.Query.resourceList, resourceList_findOne],
        put: [ResourceListEndpoints.Mutation.resourceListUpdate, resourceList_update],
    },
    "/resourceLists": {
        get: [ResourceListEndpoints.Query.resourceLists, resourceList_findMany],
    },
    "/resourceList": {
        post: [ResourceListEndpoints.Mutation.resourceListCreate, resourceList_create],
    },
});
