import { StatsSiteSearchInput, StatsSiteSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsStatsSite = {
    findMany: ApiEndpoint<StatsSiteSearchInput, StatsSiteSearchResult>;
}

const objectType = "StatsSite";
export const statsSite: EndpointsStatsSite = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
