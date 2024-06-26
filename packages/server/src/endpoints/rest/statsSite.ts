import { statsSite_findMany } from "../generated";
import { StatsSiteEndpoints } from "../logic/statsSite";
import { setupRoutes } from "./base";

export const StatsSiteRest = setupRoutes({
    "/stats/site": {
        get: [StatsSiteEndpoints.Query.statsSite, statsSite_findMany],
    },
});
