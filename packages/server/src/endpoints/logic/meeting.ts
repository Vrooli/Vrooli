import { type FindByPublicIdInput, type Meeting, type MeetingCreateInput, type MeetingSearchInput, type MeetingSearchResult, type MeetingUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsMeeting = {
    findOne: ApiEndpoint<FindByPublicIdInput, Meeting>;
    findMany: ApiEndpoint<MeetingSearchInput, MeetingSearchResult>;
    createOne: ApiEndpoint<MeetingCreateInput, Meeting>;
    updateOne: ApiEndpoint<MeetingUpdateInput, Meeting>;
}

export const meeting: EndpointsMeeting = createStandardCrudEndpoints({
    objectType: "Meeting",
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
