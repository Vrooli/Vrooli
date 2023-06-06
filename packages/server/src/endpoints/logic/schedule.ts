import { FindVersionInput, Schedule, ScheduleCreateInput, ScheduleSearchInput, ScheduleUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsSchedule = {
    Query: {
        schedule: GQLEndpoint<FindVersionInput, FindOneResult<Schedule>>;
        schedules: GQLEndpoint<ScheduleSearchInput, FindManyResult<Schedule>>;
    },
    Mutation: {
        scheduleCreate: GQLEndpoint<ScheduleCreateInput, CreateOneResult<Schedule>>;
        scheduleUpdate: GQLEndpoint<ScheduleUpdateInput, UpdateOneResult<Schedule>>;
    }
}

const objectType = "Schedule";
export const ScheduleEndpoints: EndpointsSchedule = {
    Query: {
        schedule: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        schedules: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        scheduleCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        scheduleUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
