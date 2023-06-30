import { RunRoutineInput, RunRoutineInputSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { FindManyResult, GQLEndpoint } from "../../types";

export type EndpointsRunRoutineInput = {
    Query: {
        runRoutineInputs: GQLEndpoint<RunRoutineInputSearchInput, FindManyResult<RunRoutineInput>>;
    },
}

const objectType = "RunRoutineInput";
export const RunRoutineInputEndpoints: EndpointsRunRoutineInput = {
    Query: {
        runRoutineInputs: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
