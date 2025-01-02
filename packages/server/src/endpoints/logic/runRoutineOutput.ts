import { RunRoutineOutput, RunRoutineOutputSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult } from "../../types";

export type EndpointsRunRoutineOutput = {
    findMany: ApiEndpoint<RunRoutineOutputSearchInput, FindManyResult<RunRoutineOutput>>;
}

const objectType = "RunRoutineOutput";
export const runRoutineOutput: EndpointsRunRoutineOutput = {
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
