import { standardVersion_create, standardVersion_findMany, standardVersion_findOne, standardVersion_update } from "../generated";
import { StandardVersionEndpoints } from "../logic/standardVersion";
import { setupRoutes } from "./base";

export const StandardVersionRest = setupRoutes({
    "/standardVersion/:id": {
        get: [StandardVersionEndpoints.Query.standardVersion, standardVersion_findOne],
        put: [StandardVersionEndpoints.Mutation.standardVersionUpdate, standardVersion_update],
    },
    "/standardVersions": {
        get: [StandardVersionEndpoints.Query.standardVersions, standardVersion_findMany],
    },
    "/standardVersion": {
        post: [StandardVersionEndpoints.Mutation.standardVersionCreate, standardVersion_create],
    },
});
