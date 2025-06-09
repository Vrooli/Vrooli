import { type FindByPublicIdInput, type ResourceVersion, type ResourceVersionCreateInput, type ResourceVersionSearchInput, type ResourceVersionSearchResult, type ResourceVersionUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsResource = {
    findOne: ApiEndpoint<FindByPublicIdInput, ResourceVersion>;
    findMany: ApiEndpoint<ResourceVersionSearchInput, ResourceVersionSearchResult>;
    createOne: ApiEndpoint<ResourceVersionCreateInput, ResourceVersion>;
    updateOne: ApiEndpoint<ResourceVersionUpdateInput, ResourceVersion>;
}

//TODO favor root id and versionLabel from url. Should return specified or latest public version
export const resource: EndpointsResource = createStandardCrudEndpoints({
    objectType: "ResourceVersion",
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
            rateLimit: RateLimitPresets.MEDIUM,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
