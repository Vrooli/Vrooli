import { RunRoutineOutput, RunRoutineOutputSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult } from "../../types";

export type EndpointsRunRoutineOutput = {
    Query: {
        runRoutineOutputs: ApiEndpoint<RunRoutineOutputSearchInput, FindManyResult<RunRoutineOutput>>;
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
