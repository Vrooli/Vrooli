import { type FindByIdInput, type Tag, type TagCreateInput, type TagSearchInput, type TagSearchResult, type TagUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsTag = {
    findOne: ApiEndpoint<FindByIdInput, Tag>;
    findMany: ApiEndpoint<TagSearchInput, TagSearchResult>;
    createOne: ApiEndpoint<TagCreateInput, Tag>;
    updateOne: ApiEndpoint<TagUpdateInput, Tag>;
}

export const tag: EndpointsTag = createStandardCrudEndpoints({
    objectType: "Tag",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            useEmbeddings: true,
        },
        createOne: {
            rateLimit: RateLimitPresets.MEDIUM,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.MEDIUM,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
