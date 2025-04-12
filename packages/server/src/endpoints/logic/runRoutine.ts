import { FindByIdInput, RunRoutine, RunRoutineCreateInput, RunRoutineSearchInput, RunRoutineSearchResult, RunRoutineUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsRunRoutine = {
    findOne: ApiEndpoint<FindByIdInput, RunRoutine>;
    findMany: ApiEndpoint<RunRoutineSearchInput, RunRoutineSearchResult>;
    createOne: ApiEndpoint<RunRoutineCreateInput, RunRoutine>;
    updateOne: ApiEndpoint<RunRoutineUpdateInput, RunRoutine>;
}

const objectType = "RunRoutine";
export const runRoutine: EndpointsRunRoutine = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
