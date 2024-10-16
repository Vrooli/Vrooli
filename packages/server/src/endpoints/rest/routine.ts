import { routine_create, routine_findMany, routine_findOne, routine_update } from "../generated";
import { RoutineEndpoints } from "../logic/routine";
import { setupRoutes } from "./base";

export const RoutineRest = setupRoutes({
    "/routine/:id": {
        get: [RoutineEndpoints.Query.routine, routine_findOne],
        put: [RoutineEndpoints.Mutation.routineUpdate, routine_update],
    },
    "/routines": {
        get: [RoutineEndpoints.Query.routines, routine_findMany],
    },
    "/routine": {
        post: [RoutineEndpoints.Mutation.routineCreate, routine_create],
    },
});
