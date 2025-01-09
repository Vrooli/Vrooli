import { FindByIdInput, Tag, TagCreateInput, TagSearchInput, TagSearchResult, TagUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyWithEmbeddingsHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsTag = {
    findOne: ApiEndpoint<FindByIdInput, Tag>;
    findMany: ApiEndpoint<TagSearchInput, TagSearchResult>;
    createOne: ApiEndpoint<TagCreateInput, Tag>;
    updateOne: ApiEndpoint<TagUpdateInput, Tag>;
}

const objectType = "Tag";
export const tag: EndpointsTag = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyWithEmbeddingsHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
