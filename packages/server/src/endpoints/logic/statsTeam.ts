import { StatsTeamSearchInput, StatsTeamSearchResult, VisibilityType } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsStatsTeam = {
    findMany: ApiEndpoint<StatsTeamSearchInput, StatsTeamSearchResult>;
}

const objectType = "StatsTeam";
export const statsTeam: EndpointsStatsTeam = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.OwnOrPublic });
    },
};
