import { StatsQuizSearchInput, StatsQuizSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsQuiz = {
    Query: {
        statsQuiz: GQLEndpoint<StatsQuizSearchInput, StatsQuizSearchResult>;
    },
}

const objectType = "StatsQuiz";
export const StatsQuizEndpoints: EndpointsStatsQuiz = {
    Query: {
        statsQuiz: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
