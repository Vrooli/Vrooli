import { StatsApiSearchInput, StatsApiSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsStatsApi = {
    findMany: ApiEndpoint<StatsApiSearchInput, StatsApiSearchResult>;
}

const objectType = "StatsApi";
export const statsApi: EndpointsStatsApi = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readManyHelper({ info, input, objectType, req });
    },
};
