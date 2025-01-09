import { StatsQuizSearchInput, StatsQuizSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsQuiz = {
    findMany: ApiEndpoint<StatsQuizSearchInput, StatsQuizSearchResult>;
}

const objectType = "StatsQuiz";
export const statsQuiz: EndpointsStatsQuiz = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
