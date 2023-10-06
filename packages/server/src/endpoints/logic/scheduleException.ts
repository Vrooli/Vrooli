import { FindByIdInput, ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionSearchInput, ScheduleExceptionUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsScheduleException = {
    Query: {
        scheduleException: GQLEndpoint<FindByIdInput, FindOneResult<ScheduleException>>;
        scheduleExceptions: GQLEndpoint<ScheduleExceptionSearchInput, FindManyResult<ScheduleException>>;
    },
    Mutation: {
        scheduleExceptionCreate: GQLEndpoint<ScheduleExceptionCreateInput, CreateOneResult<ScheduleException>>;
        scheduleExceptionUpdate: GQLEndpoint<ScheduleExceptionUpdateInput, UpdateOneResult<ScheduleException>>;
    }
}

const objectType = "ScheduleException";
export const ScheduleExceptionEndpoints: EndpointsScheduleException = {
    Query: {
        scheduleException: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        scheduleExceptions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        scheduleExceptionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        scheduleExceptionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
