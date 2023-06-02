import { statsApi_findMany } from "@local/shared";
import { StatsApiEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsApiRest = setupRoutes({
    "/stats/api": {
        get: [StatsApiEndpoints.Query.statsApi, statsApi_findMany],
    },
});
