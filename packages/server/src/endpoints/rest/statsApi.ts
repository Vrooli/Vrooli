import { statsApi_findMany } from "../generated";
import { StatsApiEndpoints } from "../logic/statsApi";
import { setupRoutes } from "./base";

export const StatsApiRest = setupRoutes({
    "/stats/api": {
        get: [StatsApiEndpoints.Query.statsApi, statsApi_findMany],
    },
});
