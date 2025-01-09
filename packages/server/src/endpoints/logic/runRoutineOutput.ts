import { RunRoutineOutputSearchInput, RunRoutineOutputSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsRunRoutineOutput = {
    findMany: ApiEndpoint<RunRoutineOutputSearchInput, RunRoutineOutputSearchResult>;
}

const objectType = "RunRoutineOutput";
export const runRoutineOutput: EndpointsRunRoutineOutput = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
