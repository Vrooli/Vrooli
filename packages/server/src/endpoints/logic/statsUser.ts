import { StatsUserSearchInput, StatsUserSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsUser = {
    findMany: ApiEndpoint<StatsUserSearchInput, StatsUserSearchResult>;
}

const objectType = "StatsUser";
export const statsUser: EndpointsStatsUser = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
