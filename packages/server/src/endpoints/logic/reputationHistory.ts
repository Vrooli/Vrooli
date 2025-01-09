import { FindByIdInput, ReputationHistory, ReputationHistorySearchInput, ReputationHistorySearchResult } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

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
