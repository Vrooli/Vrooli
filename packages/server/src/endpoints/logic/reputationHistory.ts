import { FindByIdInput, ReputationHistory, ReputationHistorySearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../../types";

export type EndpointsReputationHistory = {
    Query: {
        reputationHistory: GQLEndpoint<FindByIdInput, FindOneResult<ReputationHistory>>;
        reputationHistories: GQLEndpoint<ReputationHistorySearchInput, FindManyResult<ReputationHistory>>;
    },
}

const objectType = "ReputationHistory";
export const ReputationHistoryEndpoints: EndpointsReputationHistory = {
    Query: {
        reputationHistory: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        reputationHistories: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
