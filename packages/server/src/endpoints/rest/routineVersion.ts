import { routineVersion_create, routineVersion_findMany, routineVersion_findOne, routineVersion_update } from "../generated";
import { RoutineVersionEndpoints } from "../logic/routineVersion";
import { setupRoutes } from "./base";

export const RoutineVersionRest = setupRoutes({
    "/routineVersion/:id": {
        get: [RoutineVersionEndpoints.Query.routineVersion, routineVersion_findOne],
        put: [RoutineVersionEndpoints.Mutation.routineVersionUpdate, routineVersion_update],
    },
    "/routineVersions": {
        get: [RoutineVersionEndpoints.Query.routineVersions, routineVersion_findMany],
    },
    "/routineVersion": {
        post: [RoutineVersionEndpoints.Mutation.routineVersionCreate, routineVersion_create],
    },
});
