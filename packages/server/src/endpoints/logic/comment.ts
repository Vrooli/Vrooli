import { Comment, CommentCreateInput, CommentSearchInput, CommentSearchResult, CommentUpdateInput, FindByIdInput } from "@local/shared";
import { createHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CommentModel } from "../../models/base";
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
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        comments: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return CommentModel.query.searchNested(prisma, req, input, info);
        },
    },
    Mutation: {
        commentCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        commentUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
