import { StatsSiteSearchInput, StatsSiteSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsSite = {
    findMany: ApiEndpoint<StatsSiteSearchInput, StatsSiteSearchResult>;
}

const objectType = "StatsSite";
export const statsSite: EndpointsStatsSite = {
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
