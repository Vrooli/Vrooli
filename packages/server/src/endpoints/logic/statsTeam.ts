import { type StatsTeamSearchInput, type StatsTeamSearchResult, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsStatsTeam = {
    findMany: ApiEndpoint<StatsTeamSearchInput, StatsTeamSearchResult>;
}

export const statsTeam: EndpointsStatsTeam = createStandardCrudEndpoints({
    objectType: "StatsTeam",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            visibility: VisibilityType.OwnOrPublic,
        },
    },
});
