import { type StatsUserSearchInput, type StatsUserSearchResult, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsStatsUser = {
    findMany: ApiEndpoint<StatsUserSearchInput, StatsUserSearchResult>;
}

export const statsUser: EndpointsStatsUser = createStandardCrudEndpoints({
    objectType: "StatsUser",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            visibility: VisibilityType.OwnOrPublic,
        },
    },
});
