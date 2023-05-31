import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from "@local/shared";
import { createHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
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
        reminderListCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        reminderListUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
