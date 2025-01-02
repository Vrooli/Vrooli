import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, UpdateOneResult } from "../../types";

export type EndpointsReminderList = {
    createOne: ApiEndpoint<ReminderListCreateInput, CreateOneResult<ReminderList>>;
    updateOne: ApiEndpoint<ReminderListUpdateInput, UpdateOneResult<ReminderList>>;
}

const objectType = "ReminderList";
export const reminderList: EndpointsReminderList = {
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
