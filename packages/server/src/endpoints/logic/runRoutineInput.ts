import { RunRoutineInput, RunRoutineInputSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult } from "../../types";

export type EndpointsRunRoutineInput = {
    findMany: ApiEndpoint<RunRoutineInputSearchInput, FindManyResult<RunRoutineInput>>;
}

const objectType = "RunRoutineInput";
export const runRoutineInput: EndpointsRunRoutineInput = {
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
