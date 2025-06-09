import { type Chat, type ChatCreateInput, type ChatSearchInput, type ChatSearchResult, type ChatUpdateInput, type FindByPublicIdInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsChat = {
    findOne: ApiEndpoint<FindByPublicIdInput, Chat>;
    findMany: ApiEndpoint<ChatSearchInput, ChatSearchResult>;
    createOne: ApiEndpoint<ChatCreateInput, Chat>;
    updateOne: ApiEndpoint<ChatUpdateInput, Chat>;
}

export const chat: EndpointsChat = createStandardCrudEndpoints({
    objectType: "Chat",
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
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
