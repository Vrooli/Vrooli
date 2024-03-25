import { StatsOrganizationSearchInput, StatsOrganizationSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsOrganization = {
    Query: {
        statsOrganization: GQLEndpoint<StatsOrganizationSearchInput, StatsOrganizationSearchResult>;
    },
}

const objectType = "StatsOrganization";
export const StatsOrganizationEndpoints: EndpointsStatsOrganization = {
    Query: {
        statsOrganization: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
