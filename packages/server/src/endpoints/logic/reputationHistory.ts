import { FindByIdInput, ReputationHistory, ReputationHistorySearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult, FindOneResult } from "../../types";

export type EndpointsReputationHistory = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<ReputationHistory>>;
    findMany: ApiEndpoint<ReputationHistorySearchInput, FindManyResult<ReputationHistory>>;
}

const objectType = "ReputationHistory";
export const reputationHistory: EndpointsReputationHistory = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
