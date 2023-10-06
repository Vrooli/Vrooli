import { FindVersionInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionSearchInput, RoutineVersionUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsRoutineVersion = {
    Query: {
        routineVersion: GQLEndpoint<FindVersionInput, FindOneResult<RoutineVersion>>;
        routineVersions: GQLEndpoint<RoutineVersionSearchInput, FindManyResult<RoutineVersion>>;
    },
    Mutation: {
        routineVersionCreate: GQLEndpoint<RoutineVersionCreateInput, CreateOneResult<RoutineVersion>>;
        routineVersionUpdate: GQLEndpoint<RoutineVersionUpdateInput, UpdateOneResult<RoutineVersion>>;
    }
}

const objectType = "RoutineVersion";
export const RoutineVersionEndpoints: EndpointsRoutineVersion = {
    Query: {
        routineVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        routineVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        routineVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        routineVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
