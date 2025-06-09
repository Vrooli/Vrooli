import { type Comment, type CommentCreateInput, type CommentSearchInput, type CommentSearchResult, type CommentUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { CommentModel } from "../../models/base/comment.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsComment = {
    findOne: ApiEndpoint<FindByIdInput, Comment>;
    findMany: ApiEndpoint<CommentSearchInput, CommentSearchResult>;
    createOne: ApiEndpoint<CommentCreateInput, Comment>;
    updateOne: ApiEndpoint<CommentUpdateInput, Comment>;
}

export const comment: EndpointsComment = createStandardCrudEndpoints({
    objectType: "Comment",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            customImplementation: async ({ input, req, info }) => {
                return CommentModel.query.searchNested(req, input, info);
            },
        },
        createOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
