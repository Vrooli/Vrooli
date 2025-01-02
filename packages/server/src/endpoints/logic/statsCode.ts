import { StatsCodeSearchInput, StatsCodeSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsCode = {
    findMany: ApiEndpoint<StatsCodeSearchInput, StatsCodeSearchResult>;
}

const objectType = "StatsCode";
export const statsCode: EndpointsStatsCode = {
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
