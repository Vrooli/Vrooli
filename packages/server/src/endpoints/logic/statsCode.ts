import { StatsCodeSearchInput, StatsCodeSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsStatsCode = {
    findMany: ApiEndpoint<StatsCodeSearchInput, StatsCodeSearchResult>;
}

const objectType = "StatsCode";
export const statsCode: EndpointsStatsCode = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
