import { type StatsResourceSearchInput, type StatsResourceSearchResult, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsStatsResource = {
    findMany: ApiEndpoint<StatsResourceSearchInput, StatsResourceSearchResult>;
}

export const statsResource: EndpointsStatsResource = createStandardCrudEndpoints({
    objectType: "StatsResource",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            visibility: VisibilityType.OwnOrPublic,
        },
    },
});
