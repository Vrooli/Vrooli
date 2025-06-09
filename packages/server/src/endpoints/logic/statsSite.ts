import { type StatsSiteSearchInput, type StatsSiteSearchResult } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsStatsSite = {
    findMany: ApiEndpoint<StatsSiteSearchInput, StatsSiteSearchResult>;
}

export const statsSite: EndpointsStatsSite = createStandardCrudEndpoints({
    objectType: "StatsSite",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
    },
});
