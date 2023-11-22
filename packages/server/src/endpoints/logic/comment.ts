import { Comment, CommentCreateInput, CommentSearchInput, CommentSearchResult, CommentUpdateInput, FindByIdInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CommentModel } from "../../models/base/comment";
import { CreateOneResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";


export type EndpointsComment = {
    Query: {
        comment: GQLEndpoint<FindByIdInput, FindOneResult<Comment>>;
        comments: GQLEndpoint<CommentSearchInput, CommentSearchResult>;
    },
    Mutation: {
        commentCreate: GQLEndpoint<CommentCreateInput, CreateOneResult<Comment>>;
        commentUpdate: GQLEndpoint<CommentUpdateInput, UpdateOneResult<Comment>>;
    }
}

const objectType = "Comment";
export const CommentEndpoints: EndpointsComment = {
    Query: {
        comment: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        comments: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return CommentModel.query.searchNested(prisma, req, input, info);
        },
    },
    Mutation: {
        commentCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        commentUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
