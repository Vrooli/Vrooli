import { StatsQuizSearchInput, StatsQuizSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsQuiz = {
    Query: {
        statsQuiz: GQLEndpoint<StatsQuizSearchInput, StatsQuizSearchResult>;
    },
}

const objectType = "StatsQuiz";
export const StatsQuizEndpoints: EndpointsStatsQuiz = {
    Query: {
        statsQuiz: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
