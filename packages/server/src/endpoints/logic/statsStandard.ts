import { StatsStandardSearchInput, StatsStandardSearchResult, VisibilityType } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsStatsStandard = {
    findMany: ApiEndpoint<StatsStandardSearchInput, StatsStandardSearchResult>;
}

const objectType = "StatsStandard";
export const statsStandard: EndpointsStatsStandard = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.OwnOrPublic });
    },
};
