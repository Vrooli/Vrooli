import { type ChatParticipant, type ChatParticipantSearchInput, type ChatParticipantSearchResult, type ChatParticipantUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsChatParticipant = {
    findOne: ApiEndpoint<FindByIdInput, ChatParticipant>;
    findMany: ApiEndpoint<ChatParticipantSearchInput, ChatParticipantSearchResult>;
    updateOne: ApiEndpoint<ChatParticipantUpdateInput, ChatParticipant>;
}

export const chatParticipant: EndpointsChatParticipant = createStandardCrudEndpoints({
    objectType: "ChatParticipant",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
