import { runRoutineInput_findMany } from "../generated";
import { RunRoutineInputEndpoints } from "../logic/runRoutineInput";
import { setupRoutes } from "./base";

export const RunRoutineInputRest = setupRoutes({
    "/runRoutineInputs": {
        get: [RunRoutineInputEndpoints.Query.runRoutineInputs, runRoutineInput_findMany],
    },
});
