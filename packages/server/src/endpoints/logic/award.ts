import { type AwardSearchInput, type AwardSearchResult, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsAward = {
    findMany: ApiEndpoint<AwardSearchInput, AwardSearchResult>;
}

export const award: EndpointsAward = createStandardCrudEndpoints({
    objectType: "Award",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
            visibility: VisibilityType.Own,
        },
    },
});
