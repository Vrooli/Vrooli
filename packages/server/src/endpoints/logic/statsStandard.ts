import { StatsStandardSearchInput, StatsStandardSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsStandard = {
    Query: {
        statsStandard: GQLEndpoint<StatsStandardSearchInput, StatsStandardSearchResult>;
    },
}

const objectType = "StatsStandard";
export const StatsStandardEndpoints: EndpointsStatsStandard = {
    Query: {
        statsStandard: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
