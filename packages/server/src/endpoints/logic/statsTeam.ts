import { StatsTeamSearchInput, StatsTeamSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsTeam = {
    Query: {
        statsTeam: GQLEndpoint<StatsTeamSearchInput, StatsTeamSearchResult>;
    },
}

const objectType = "StatsTeam";
export const StatsTeamEndpoints: EndpointsStatsTeam = {
    Query: {
        statsTeam: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
