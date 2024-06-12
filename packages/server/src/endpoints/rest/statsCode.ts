import { statsCode_findMany } from "../generated";
import { StatsCodeEndpoints } from "../logic/statsCode";
import { setupRoutes } from "./base";

export const StatsCodeRest = setupRoutes({
    "/stats/code": {
        get: [StatsCodeEndpoints.Query.statsCode, statsCode_findMany],
    },
});

