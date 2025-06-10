import { type FindByIdInput, type ReputationHistory, type ReputationHistorySearchInput, type ReputationHistorySearchResult } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsReputationHistory = {
    findOne: ApiEndpoint<FindByIdInput, ReputationHistory>;
    findMany: ApiEndpoint<ReputationHistorySearchInput, ReputationHistorySearchResult>;
}

export const reputationHistory: EndpointsReputationHistory = createStandardCrudEndpoints({
    objectType: "ReputationHistory",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
    },
});
