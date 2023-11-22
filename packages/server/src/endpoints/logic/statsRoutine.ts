import { StatsRoutineSearchInput, StatsRoutineSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsStatsRoutine = {
    Query: {
        statsRoutine: GQLEndpoint<StatsRoutineSearchInput, StatsRoutineSearchResult>;
    },
}

const objectType = "StatsRoutine";
export const StatsRoutineEndpoints: EndpointsStatsRoutine = {
    Query: {
        statsRoutine: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
