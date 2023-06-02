import { statsStandard_findMany } from "@local/shared";
import { StatsStandardEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsStandardRest = setupRoutes({
    "/stats/standard": {
        get: [StatsStandardEndpoints.Query.statsStandard, statsStandard_findMany],
    },
});
