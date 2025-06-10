import { type ReminderList, type ReminderListCreateInput, type ReminderListUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsReminderList = {
    createOne: ApiEndpoint<ReminderListCreateInput, ReminderList>;
    updateOne: ApiEndpoint<ReminderListUpdateInput, ReminderList>;
}

export const reminderList: EndpointsReminderList = createStandardCrudEndpoints({
    objectType: "ReminderList",
    endpoints: {
        createOne: {
            rateLimit: RateLimitPresets.STRICT,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.MEDIUM,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
