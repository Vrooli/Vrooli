import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsReminderList = {
    createOne: ApiEndpoint<ReminderListCreateInput, ReminderList>;
    updateOne: ApiEndpoint<ReminderListUpdateInput, ReminderList>;
}

const objectType = "ReminderList";
export const reminderList: EndpointsReminderList = {
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
