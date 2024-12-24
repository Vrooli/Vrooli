import { StatsTeamSearchInput, StatsTeamSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsTeam = {
    Query: {
        statsTeam: ApiEndpoint<StatsTeamSearchInput, StatsTeamSearchResult>;
    },
}

const objectType = "StatsTeam";
export const StatsTeamEndpoints: EndpointsStatsTeam = {
    Query: {
        statsTeam: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
