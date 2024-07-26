import { runRoutine_create, runRoutine_deleteAll, runRoutine_findMany, runRoutine_findOne, runRoutine_update } from "../generated";
import { RunRoutineEndpoints } from "../logic/runRoutine";
import { setupRoutes } from "./base";

export const RunRoutineRest = setupRoutes({
    "/runRoutine/:id": {
        get: [RunRoutineEndpoints.Query.runRoutine, runRoutine_findOne],
        put: [RunRoutineEndpoints.Mutation.runRoutineUpdate, runRoutine_update],
    },
    "/runRoutines": {
        get: [RunRoutineEndpoints.Query.runRoutines, runRoutine_findMany],
    },
    "/runRoutine": {
        post: [RunRoutineEndpoints.Mutation.runRoutineCreate, runRoutine_create],
    },
    "/runRoutine/deleteAll": {
        delete: [RunRoutineEndpoints.Mutation.runRoutineDeleteAll, runRoutine_deleteAll],
    },
});
