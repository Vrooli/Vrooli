import { StatsSiteSearchInput, StatsSiteSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsSite = {
    Query: {
        statsSite: ApiEndpoint<StatsSiteSearchInput, StatsSiteSearchResult>;
    },
}

const objectType = "StatsSite";
export const StatsSiteEndpoints: EndpointsStatsSite = {
    Query: {
        statsSite: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
