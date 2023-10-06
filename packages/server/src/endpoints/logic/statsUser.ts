import { StatsUserSearchInput, StatsUserSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsUser = {
    Query: {
        statsUser: GQLEndpoint<StatsUserSearchInput, StatsUserSearchResult>;
    },
}

const objectType = "StatsUser";
export const StatsUserEndpoints: EndpointsStatsUser = {
    Query: {
        statsUser: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
