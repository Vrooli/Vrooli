import { RunRoutineInputSearchInput, RunRoutineInputSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsRunRoutineInput = {
    findMany: ApiEndpoint<RunRoutineInputSearchInput, RunRoutineInputSearchResult>;
}

const objectType = "RunRoutineInput";
export const runRoutineInput: EndpointsRunRoutineInput = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
