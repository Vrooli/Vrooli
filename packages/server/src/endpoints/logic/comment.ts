import { Comment, CommentCreateInput, CommentSearchInput, CommentSearchResult, CommentUpdateInput, FindByIdInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CommentModel } from "../../models/base/comment";
import { ApiEndpoint } from "../../types";

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
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return CommentModel.query.searchNested(req, input, info);
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
