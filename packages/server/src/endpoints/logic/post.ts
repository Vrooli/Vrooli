import { FindByIdInput, Post, PostCreateInput, PostSearchInput, PostUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsPost = {
    Query: {
        post: GQLEndpoint<FindByIdInput, FindOneResult<Post>>;
        posts: GQLEndpoint<PostSearchInput, FindManyResult<Post>>;
    },
    Mutation: {
        postCreate: GQLEndpoint<PostCreateInput, CreateOneResult<Post>>;
        postUpdate: GQLEndpoint<PostUpdateInput, UpdateOneResult<Post>>;
    }
}

const objectType = "Post";
export const PostEndpoints: EndpointsPost = {
    Query: {
        post: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        posts: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        postCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        postUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
