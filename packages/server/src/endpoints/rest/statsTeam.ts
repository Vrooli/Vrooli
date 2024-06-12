import { statsTeam_findMany } from "../generated";
import { StatsTeamEndpoints } from "../logic/statsTeam";
import { setupRoutes } from "./base";

export const StatsTeamRest = setupRoutes({
    "/stats/team": {
        get: [StatsTeamEndpoints.Query.statsTeam, statsTeam_findMany],
    },
});
