import { FindByIdInput, Routine, RoutineCreateInput, RoutineSearchInput, RoutineUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsRoutine = {
    Query: {
        routine: GQLEndpoint<FindByIdInput, FindOneResult<Routine>>;
        routines: GQLEndpoint<RoutineSearchInput, FindManyResult<Routine>>;
    },
    Mutation: {
        routineCreate: GQLEndpoint<RoutineCreateInput, CreateOneResult<Routine>>;
        routineUpdate: GQLEndpoint<RoutineUpdateInput, UpdateOneResult<Routine>>;
    }
}

const objectType = "Routine";
export const RoutineEndpoints: EndpointsRoutine = {
    Query: {
        routine: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        routines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        routineCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        routineUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
