import { Comment, CommentCreateInput, CommentSearchInput, CommentSearchResult, CommentUpdateInput, FindByIdInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { CommentModel } from "../../models/base/comment.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsComment = {
    findOne: ApiEndpoint<FindByIdInput, Comment>;
    findMany: ApiEndpoint<CommentSearchInput, CommentSearchResult>;
    createOne: ApiEndpoint<CommentCreateInput, Comment>;
    updateOne: ApiEndpoint<CommentUpdateInput, Comment>;
}

const objectType = "Comment";
export const comment: EndpointsComment = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return CommentModel.query.searchNested(req, input, info);
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
};
