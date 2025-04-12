import { StatsUserSearchInput, StatsUserSearchResult, VisibilityType } from "@local/shared";
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
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.OwnOrPublic });
    },
};
