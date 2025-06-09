import { type StatsProjectSearchInput, type StatsProjectSearchResult, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsStatsProject = {
    findMany: ApiEndpoint<StatsProjectSearchInput, StatsProjectSearchResult>;
}

export const statsProject: EndpointsStatsProject = createStandardCrudEndpoints({
    objectType: "StatsProject",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            visibility: VisibilityType.OwnOrPublic,
        },
    },
});
