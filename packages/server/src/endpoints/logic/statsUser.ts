import { StatsUserSearchInput, StatsUserSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsStatsUser = {
    findMany: ApiEndpoint<StatsUserSearchInput, StatsUserSearchResult>;
}

const objectType = "StatsUser";
export const statsUser: EndpointsStatsUser = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readManyHelper({ info, input, objectType, req });
    },
};
