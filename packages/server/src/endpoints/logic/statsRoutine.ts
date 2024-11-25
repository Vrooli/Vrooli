import { StatsRoutineSearchInput, StatsRoutineSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsRoutine = {
    Query: {
        statsRoutine: GQLEndpoint<StatsRoutineSearchInput, StatsRoutineSearchResult>;
    },
}

const objectType = "StatsRoutine";
export const StatsRoutineEndpoints: EndpointsStatsRoutine = {
    Query: {
        statsRoutine: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
