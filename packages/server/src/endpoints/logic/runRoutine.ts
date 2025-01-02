import { FindByIdInput, RunRoutine, RunRoutineCreateInput, RunRoutineSearchInput, RunRoutineUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsRunRoutine = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<RunRoutine>>;
    findMany: ApiEndpoint<RunRoutineSearchInput, FindManyResult<RunRoutine>>;
    createOne: ApiEndpoint<RunRoutineCreateInput, CreateOneResult<RunRoutine>>;
    updateOne: ApiEndpoint<RunRoutineUpdateInput, UpdateOneResult<RunRoutine>>;
}

const objectType = "RunRoutine";
export const runRoutine: EndpointsRunRoutine = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
