import { runRoutineInput_findMany } from "@local/shared";
import { RunRoutineInputEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const RunRoutineInputRest = setupRoutes({
    "/runRoutineInputs": {
        get: [RunRoutineInputEndpoints.Query.runRoutineInputs, runRoutineInput_findMany],
    },
});
