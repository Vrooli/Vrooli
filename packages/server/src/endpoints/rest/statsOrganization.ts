import { statsOrganization_findMany } from "@local/shared";
import { StatsOrganizationEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsOrganizationRest = setupRoutes({
    "/stats/organization": {
        get: [StatsOrganizationEndpoints.Query.statsOrganization, statsOrganization_findMany],
    },
});
