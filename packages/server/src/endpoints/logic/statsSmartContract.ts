import { StatsSmartContractSearchInput, StatsSmartContractSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsSmartContract = {
    Query: {
        statsSmartContract: GQLEndpoint<StatsSmartContractSearchInput, StatsSmartContractSearchResult>;
    },
}

const objectType = "StatsSmartContract";
export const StatsSmartContractEndpoints: EndpointsStatsSmartContract = {
    Query: {
        statsSmartContract: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
