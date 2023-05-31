import { FindByIdInput, ReputationHistory, ReputationHistorySearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions";
import { rateLimit } from "../../middleware";
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
        reputationHistory: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        reputationHistories: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
