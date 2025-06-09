import { type RunIOSearchInput, type RunIOSearchResult } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsRunIO = {
    findMany: ApiEndpoint<RunIOSearchInput, RunIOSearchResult>;
}

export const runIO: EndpointsRunIO = createStandardCrudEndpoints({
    objectType: "RunIO",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
    },
});
