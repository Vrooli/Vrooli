import { statsRoutine_findMany } from "@local/shared";
import { StatsRoutineEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsRoutineRest = setupRoutes({
    "/stats/routine": {
        get: [StatsRoutineEndpoints.Query.statsRoutine, statsRoutine_findMany],
    },
});
