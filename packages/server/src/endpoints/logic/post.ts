import { FindByIdInput, Post, PostCreateInput, PostSearchInput, PostUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
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
        post: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        posts: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        postCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        postUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
