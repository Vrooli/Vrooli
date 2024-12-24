import { FindByIdInput, Post, PostCreateInput, PostSearchInput, PostUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsPost = {
    Query: {
        post: ApiEndpoint<FindByIdInput, FindOneResult<Post>>;
        posts: ApiEndpoint<PostSearchInput, FindManyResult<Post>>;
    },
    Mutation: {
        postCreate: ApiEndpoint<PostCreateInput, CreateOneResult<Post>>;
        postUpdate: ApiEndpoint<PostUpdateInput, UpdateOneResult<Post>>;
    }
}

const objectType = "Post";
export const PostEndpoints: EndpointsPost = {
    Query: {
        post: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        posts: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        postCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        postUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
