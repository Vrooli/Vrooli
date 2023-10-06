import { FindByIdInput, Reminder, ReminderCreateInput, ReminderSearchInput, ReminderUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsReminder = {
    Query: {
        reminder: GQLEndpoint<FindByIdInput, FindOneResult<Reminder>>;
        reminders: GQLEndpoint<ReminderSearchInput, FindManyResult<Reminder>>;
    },
    Mutation: {
        reminderCreate: GQLEndpoint<ReminderCreateInput, CreateOneResult<Reminder>>;
        reminderUpdate: GQLEndpoint<ReminderUpdateInput, UpdateOneResult<Reminder>>;
    }
}

const objectType = "Reminder";
export const ReminderEndpoints: EndpointsReminder = {
    Query: {
        reminder: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        reminders: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        reminderCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        reminderUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
