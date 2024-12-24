import { StatsStandardSearchInput, StatsStandardSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsStandard = {
    Query: {
        statsStandard: ApiEndpoint<StatsStandardSearchInput, StatsStandardSearchResult>;
    },
}

const objectType = "StatsStandard";
export const StatsStandardEndpoints: EndpointsStatsStandard = {
    Query: {
        statsStandard: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
