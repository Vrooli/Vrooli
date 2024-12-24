import { StatsUserSearchInput, StatsUserSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsUser = {
    Query: {
        statsUser: ApiEndpoint<StatsUserSearchInput, StatsUserSearchResult>;
    },
}

const objectType = "StatsUser";
export const StatsUserEndpoints: EndpointsStatsUser = {
    Query: {
        statsUser: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
