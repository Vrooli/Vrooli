import { FindByIdInput, Reminder, ReminderCreateInput, ReminderSearchInput, ReminderUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
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
        reminder: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        reminders: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        reminderCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        reminderUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
