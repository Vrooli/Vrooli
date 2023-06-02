import { StatsApiSearchInput, StatsApiSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsApi = {
    Query: {
        statsApi: GQLEndpoint<StatsApiSearchInput, StatsApiSearchResult>;
    },
}

const objectType = "StatsApi";
export const StatsApiEndpoints: EndpointsStatsApi = {
    Query: {
        statsApi: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
