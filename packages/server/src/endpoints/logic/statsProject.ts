import { StatsProjectSearchInput, StatsProjectSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsProject = {
    Query: {
        statsProject: GQLEndpoint<StatsProjectSearchInput, StatsProjectSearchResult>;
    },
}

const objectType = "StatsProject";
export const StatsProjectEndpoints: EndpointsStatsProject = {
    Query: {
        statsProject: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
