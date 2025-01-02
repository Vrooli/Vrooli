import { StatsStandardSearchInput, StatsStandardSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsStandard = {
    findMany: ApiEndpoint<StatsStandardSearchInput, StatsStandardSearchResult>;
}

const objectType = "StatsStandard";
export const statsStandard: EndpointsStatsStandard = {
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
