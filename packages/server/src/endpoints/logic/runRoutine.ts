import { Count, FindByIdInput, RunRoutine, RunRoutineCancelInput, RunRoutineCompleteInput, RunRoutineCreateInput, RunRoutineSearchInput, RunRoutineUpdateInput, VisibilityType } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { assertRequestFrom } from "../../auth/request";
import { rateLimit } from "../../middleware/rateLimit";
import { RunRoutineModel } from "../../models/base/runRoutine";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, RecursivePartial, UpdateOneResult } from "../../types";

export type EndpointsRunRoutine = {
    Query: {
        runRoutine: GQLEndpoint<FindByIdInput, FindOneResult<RunRoutine>>;
        runRoutines: GQLEndpoint<RunRoutineSearchInput, FindManyResult<RunRoutine>>;
    },
    Mutation: {
        runRoutineCreate: GQLEndpoint<RunRoutineCreateInput, CreateOneResult<RunRoutine>>;
        runRoutineUpdate: GQLEndpoint<RunRoutineUpdateInput, UpdateOneResult<RunRoutine>>;
        runRoutineDeleteAll: GQLEndpoint<Record<string, never>, Count>;
        runRoutineComplete: GQLEndpoint<RunRoutineCompleteInput, RecursivePartial<RunRoutine>>;
        runRoutineCancel: GQLEndpoint<RunRoutineCancelInput, RecursivePartial<RunRoutine>>;
    }
}

const objectType = "RunRoutine";
export const RunRoutineEndpoints: EndpointsRunRoutine = {
    Query: {
        runRoutine: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        runRoutines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        runRoutineCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        runRoutineUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        runRoutineDeleteAll: async (_p, _d, { prisma, req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 25, req });
            return RunRoutineModel.danger.deleteAll(prisma, { __typename: "User", id: userData.id });
        },
        runRoutineComplete: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });
            return RunRoutineModel.run.complete(prisma, userData, input, info);
        },
        runRoutineCancel: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });
            return RunRoutineModel.run.cancel(prisma, userData, input, info);
        },
    },
};
