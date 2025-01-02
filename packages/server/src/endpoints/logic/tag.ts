import { FindByIdInput, Tag, TagCreateInput, TagSearchInput, TagUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyWithEmbeddingsHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsTag = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<Tag>>;
    findMany: ApiEndpoint<TagSearchInput, FindManyResult<Tag>>;
    createOne: ApiEndpoint<TagCreateInput, CreateOneResult<Tag>>;
    updateOne: ApiEndpoint<TagUpdateInput, UpdateOneResult<Tag>>;
}

const objectType = "Tag";
export const tag: EndpointsTag = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyWithEmbeddingsHelper({ info, input, objectType, req });
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
