import { StatsCodeSearchInput, StatsCodeSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsCode = {
    Query: {
        statsCode: GQLEndpoint<StatsCodeSearchInput, StatsCodeSearchResult>;
    },
}

const objectType = "StatsCode";
export const StatsCodeEndpoints: EndpointsStatsCode = {
    Query: {
        statsCode: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
