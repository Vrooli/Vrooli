import { Comment, CommentCreateInput, CommentSearchInput, CommentSearchResult, CommentUpdateInput, FindByIdInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CommentModel } from "../../models/base/comment";
import { ApiEndpoint, CreateOneResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsComment = {
    Query: {
        comment: ApiEndpoint<FindByIdInput, FindOneResult<Comment>>;
        comments: ApiEndpoint<CommentSearchInput, CommentSearchResult>;
    },
    Mutation: {
        commentCreate: ApiEndpoint<CommentCreateInput, CreateOneResult<Comment>>;
        commentUpdate: ApiEndpoint<CommentUpdateInput, UpdateOneResult<Comment>>;
    }
}

const objectType = "Comment";
export const CommentEndpoints: EndpointsComment = {
    Query: {
        comment: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        comments: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return CommentModel.query.searchNested(req, input, info);
        },
    },
    Mutation: {
        commentCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return createOneHelper({ info, input, objectType, req });
        },
        commentUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
