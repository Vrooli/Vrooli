import { StatsSiteSearchInput, StatsSiteSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsSite = {
    Query: {
        statsSite: GQLEndpoint<StatsSiteSearchInput, StatsSiteSearchResult>;
    },
}

const objectType = "StatsSite";
export const StatsSiteEndpoints: EndpointsStatsSite = {
    Query: {
        statsSite: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
