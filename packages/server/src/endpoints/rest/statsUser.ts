import { statsUser_findMany } from "../generated";
import { StatsUserEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsUserRest = setupRoutes({
    "/stats/user": {
        get: [StatsUserEndpoints.Query.statsUser, statsUser_findMany],
    },
});
