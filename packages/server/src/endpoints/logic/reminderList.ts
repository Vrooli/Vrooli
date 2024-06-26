import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsReminderList = {
    Mutation: {
        reminderListCreate: GQLEndpoint<ReminderListCreateInput, CreateOneResult<ReminderList>>;
        reminderListUpdate: GQLEndpoint<ReminderListUpdateInput, UpdateOneResult<ReminderList>>;
    }
}

const objectType = "ReminderList";
export const ReminderListEndpoints: EndpointsReminderList = {
    Mutation: {
        reminderListCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        reminderListUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
