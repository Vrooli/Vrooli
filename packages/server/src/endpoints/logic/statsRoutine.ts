import { StatsRoutineSearchInput, StatsRoutineSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsStatsRoutine = {
    findMany: ApiEndpoint<StatsRoutineSearchInput, StatsRoutineSearchResult>;
}

const objectType = "StatsRoutine";
export const statsRoutine: EndpointsStatsRoutine = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
