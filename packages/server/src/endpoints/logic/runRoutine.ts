import { Count, FindByIdInput, RunRoutine, RunRoutineCreateInput, RunRoutineSearchInput, RunRoutineUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { RunRoutineModel } from "../../models/base/runRoutine";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsRunRoutine = {
    Query: {
        runRoutine: ApiEndpoint<FindByIdInput, FindOneResult<RunRoutine>>;
        runRoutines: ApiEndpoint<RunRoutineSearchInput, FindManyResult<RunRoutine>>;
    },
    Mutation: {
        runRoutineCreate: ApiEndpoint<RunRoutineCreateInput, CreateOneResult<RunRoutine>>;
        runRoutineUpdate: ApiEndpoint<RunRoutineUpdateInput, UpdateOneResult<RunRoutine>>;
        runRoutineDeleteAll: ApiEndpoint<Record<string, never>, Count>;
    }
}

const objectType = "RunRoutine";
export const RunRoutineEndpoints: EndpointsRunRoutine = {
    Query: {
        runRoutine: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        runRoutines: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        runRoutineCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return createOneHelper({ info, input, objectType, req });
        },
        runRoutineUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        runRoutineDeleteAll: async (_p, _d, { req }) => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 25, req });
            return RunRoutineModel.danger.deleteAll({ __typename: "User", id: userData.id });
        },
    },
};
