import { RunRoutineIOSearchInput, RunRoutineIOSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsRunRoutineIO = {
    findMany: ApiEndpoint<RunRoutineIOSearchInput, RunRoutineIOSearchResult>;
}

const objectType = "RunRoutineIO";
export const runRoutineIO: EndpointsRunRoutineIO = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
