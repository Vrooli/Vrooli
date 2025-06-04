import { type FindByIdInput, type ReputationHistory, type ReputationHistorySearchInput, type ReputationHistorySearchResult } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsReputationHistory = {
    findOne: ApiEndpoint<FindByIdInput, ReputationHistory>;
    findMany: ApiEndpoint<ReputationHistorySearchInput, ReputationHistorySearchResult>;
}

const objectType = "ReputationHistory";
export const reputationHistory: EndpointsReputationHistory = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
