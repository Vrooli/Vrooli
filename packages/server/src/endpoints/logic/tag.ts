import { FindByIdInput, Tag, TagCreateInput, TagSearchInput, TagUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyWithEmbeddingsHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsTag = {
    Query: {
        tag: ApiEndpoint<FindByIdInput, FindOneResult<Tag>>;
        tags: ApiEndpoint<TagSearchInput, FindManyResult<Tag>>;
    },
    Mutation: {
        tagCreate: ApiEndpoint<TagCreateInput, CreateOneResult<Tag>>;
        tagUpdate: ApiEndpoint<TagUpdateInput, UpdateOneResult<Tag>>;
    }
}

const objectType = "Tag";
export const TagEndpoints: EndpointsTag = {
    Query: {
        tag: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        tags: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyWithEmbeddingsHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        tagCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        tagUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
