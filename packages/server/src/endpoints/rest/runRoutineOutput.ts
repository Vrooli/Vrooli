import { runRoutineOutput_findMany } from "../generated";
import { RunRoutineOutputEndpoints } from "../logic/runRoutineOutput";
import { setupRoutes } from "./base";

export const RunRoutineOutputRest = setupRoutes({
    "/runRoutineOutputs": {
        get: [RunRoutineOutputEndpoints.Query.runRoutineOutputs, runRoutineOutput_findMany],
    },
});
