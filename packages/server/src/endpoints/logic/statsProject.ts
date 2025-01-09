import { StatsProjectSearchInput, StatsProjectSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsProject = {
    findMany: ApiEndpoint<StatsProjectSearchInput, StatsProjectSearchResult>;
}

const objectType = "StatsProject";
export const statsProject: EndpointsStatsProject = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
