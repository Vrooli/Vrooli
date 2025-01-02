import { FindByIdInput, Post, PostCreateInput, PostSearchInput, PostUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsPost = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<Post>>;
    findMany: ApiEndpoint<PostSearchInput, FindManyResult<Post>>;
    createOne: ApiEndpoint<PostCreateInput, CreateOneResult<Post>>;
    updateOne: ApiEndpoint<PostUpdateInput, UpdateOneResult<Post>>;
}

const objectType = "Post";
export const post: EndpointsPost = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
