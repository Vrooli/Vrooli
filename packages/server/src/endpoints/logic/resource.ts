import { type FindByPublicIdInput, type ResourceVersion, type ResourceVersionCreateInput, type ResourceVersionSearchInput, type ResourceVersionSearchResult, type ResourceVersionUpdateInput } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsResource = {
    findOne: ApiEndpoint<FindByPublicIdInput, ResourceVersion>;
    findMany: ApiEndpoint<ResourceVersionSearchInput, ResourceVersionSearchResult>;
    createOne: ApiEndpoint<ResourceVersionCreateInput, ResourceVersion>;
    updateOne: ApiEndpoint<ResourceVersionUpdateInput, ResourceVersion>;
}

//TODO favor root id and versionLabel from url. Should return specified or latest public version
const objectType = "ResourceVersion";
export const resource: EndpointsResource = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
