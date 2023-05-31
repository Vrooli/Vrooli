import { StatsOrganizationSearchInput, StatsOrganizationSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsOrganization = {
    Query: {
        statsOrganization: GQLEndpoint<StatsOrganizationSearchInput, StatsOrganizationSearchResult>;
    },
}

const objectType = "StatsOrganization";
export const StatsOrganizationEndpoints: EndpointsStatsOrganization = {
    Query: {
        statsOrganization: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
