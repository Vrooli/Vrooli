import { StatsApiSearchInput, StatsApiSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsApi = {
    Query: {
        statsApi: GQLEndpoint<StatsApiSearchInput, StatsApiSearchResult>;
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
