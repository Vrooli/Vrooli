import { StatsApiSearchInput, StatsApiSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsApi = {
    Query: {
        statsApi: ApiEndpoint<StatsApiSearchInput, StatsApiSearchResult>;
    },
}

const objectType = "StatsApi";
export const StatsApiEndpoints: EndpointsStatsApi = {
    Query: {
        statsApi: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
