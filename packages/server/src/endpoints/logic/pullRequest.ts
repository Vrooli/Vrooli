import { type FindByPublicIdInput, type PullRequest, type PullRequestCreateInput, type PullRequestSearchInput, type PullRequestSearchResult, type PullRequestUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsPullRequest = {
    findOne: ApiEndpoint<FindByPublicIdInput, PullRequest>;
    findMany: ApiEndpoint<PullRequestSearchInput, PullRequestSearchResult>;
    createOne: ApiEndpoint<PullRequestCreateInput, PullRequest>;
    updateOne: ApiEndpoint<PullRequestUpdateInput, PullRequest>;
}

export const pullRequest: EndpointsPullRequest = createStandardCrudEndpoints({
    objectType: "PullRequest",
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
            // TODO 2 permissions for this differ from normal objects. Some fields can be updated by creator, and some by owner of object the pull request is for. Probably need to make custom endpoints like for transfers
        },
    },
});
