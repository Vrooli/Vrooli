import { FindByIdInput, Tag, TagCreateInput, TagSearchInput, TagSearchResult, TagUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyWithEmbeddingsHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

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
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readManyWithEmbeddingsHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
};
