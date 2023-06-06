import { statsRoutine_findMany } from "../generated";
import { StatsRoutineEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsRoutineRest = setupRoutes({
    "/stats/routine": {
        get: [StatsRoutineEndpoints.Query.statsRoutine, statsRoutine_findMany],
    },
});
