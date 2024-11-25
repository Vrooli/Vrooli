import { RunRoutineOutput, RunRoutineOutputSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { FindManyResult, GQLEndpoint } from "../../types";

export type EndpointsRunRoutineOutput = {
    Query: {
        runRoutineOutputs: GQLEndpoint<RunRoutineOutputSearchInput, FindManyResult<RunRoutineOutput>>;
    },
}

const objectType = "RunRoutineOutput";
export const RunRoutineOutputEndpoints: EndpointsRunRoutineOutput = {
    Query: {
        runRoutineOutputs: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
