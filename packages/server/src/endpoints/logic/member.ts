import { type FindByPublicIdInput, type Member, type MemberSearchInput, type MemberSearchResult, type MemberUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsMember = {
    findOne: ApiEndpoint<FindByPublicIdInput, Member>;
    findMany: ApiEndpoint<MemberSearchInput, MemberSearchResult>;
    updateOne: ApiEndpoint<MemberUpdateInput, Member>;
}

export const member: EndpointsMember = createStandardCrudEndpoints({
    objectType: "Member",
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
