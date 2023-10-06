import { FindByIdInput, ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceSearchInput, ScheduleRecurrenceUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsScheduleRecurrence = {
    Query: {
        scheduleRecurrence: GQLEndpoint<FindByIdInput, FindOneResult<ScheduleRecurrence>>;
        scheduleRecurrences: GQLEndpoint<ScheduleRecurrenceSearchInput, FindManyResult<ScheduleRecurrence>>;
    },
    Mutation: {
        scheduleRecurrenceCreate: GQLEndpoint<ScheduleRecurrenceCreateInput, CreateOneResult<ScheduleRecurrence>>;
        scheduleRecurrenceUpdate: GQLEndpoint<ScheduleRecurrenceUpdateInput, UpdateOneResult<ScheduleRecurrence>>;
    }
}

const objectType = "ScheduleRecurrence";
export const ScheduleRecurrenceEndpoints: EndpointsScheduleRecurrence = {
    Query: {
        scheduleRecurrence: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        scheduleRecurrences: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        scheduleRecurrenceCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        scheduleRecurrenceUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
