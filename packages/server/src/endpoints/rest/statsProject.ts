import { statsProject_findMany } from "../generated";
import { StatsProjectEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsProjectRest = setupRoutes({
    "/stats/project": {
        get: [StatsProjectEndpoints.Query.statsProject, statsProject_findMany],
    },
});
