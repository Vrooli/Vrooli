import { type FindByIdInput, type Reminder, type ReminderCreateInput, type ReminderSearchInput, type ReminderSearchResult, type ReminderUpdateInput, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsReminder = {
    findOne: ApiEndpoint<FindByIdInput, Reminder>;
    findMany: ApiEndpoint<ReminderSearchInput, ReminderSearchResult>;
    createOne: ApiEndpoint<ReminderCreateInput, Reminder>;
    updateOne: ApiEndpoint<ReminderUpdateInput, Reminder>;
}

export const reminder: EndpointsReminder = createStandardCrudEndpoints({
    objectType: "Reminder",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
            visibility: VisibilityType.Own,
        },
        createOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
