import { type FindByPublicIdInput, type Team, type TeamCreateInput, type TeamSearchInput, type TeamSearchResult, type TeamUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsTeam = {
    findOne: ApiEndpoint<FindByPublicIdInput, Team>;
    findMany: ApiEndpoint<TeamSearchInput, TeamSearchResult>;
    createOne: ApiEndpoint<TeamCreateInput, Team>;
    updateOne: ApiEndpoint<TeamUpdateInput, Team>;
}

export const team: EndpointsTeam = createStandardCrudEndpoints({
    objectType: "Team",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        createOne: {
            rateLimit: RateLimitPresets.STRICT,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
