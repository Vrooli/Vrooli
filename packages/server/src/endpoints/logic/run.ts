import { type FindByIdInput, type Run, type RunCreateInput, type RunSearchInput, type RunSearchResult, type RunUpdateInput, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsRun = {
    findOne: ApiEndpoint<FindByIdInput, Run>;
    findMany: ApiEndpoint<RunSearchInput, RunSearchResult>;
    createOne: ApiEndpoint<RunCreateInput, Run>;
    updateOne: ApiEndpoint<RunUpdateInput, Run>;
}

export const run: EndpointsRun = createStandardCrudEndpoints({
    objectType: "Run",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
            visibility: VisibilityType.Own,
        },
        createOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
