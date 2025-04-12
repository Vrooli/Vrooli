import { FindByIdInput, Standard, StandardCreateInput, StandardSearchInput, StandardSearchResult, StandardUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsStandard = {
    findOne: ApiEndpoint<FindByIdInput, Standard>;
    findMany: ApiEndpoint<StandardSearchInput, StandardSearchResult>;
    createOne: ApiEndpoint<StandardCreateInput, Standard>;
    updateOne: ApiEndpoint<StandardUpdateInput, Standard>;
}

const objectType = "Standard";
export const standard: EndpointsStandard = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
