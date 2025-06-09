import { type FindByIdInput, type NotificationSubscription, type NotificationSubscriptionCreateInput, type NotificationSubscriptionSearchInput, type NotificationSubscriptionSearchResult, type NotificationSubscriptionUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsNotificationSubscription = {
    findOne: ApiEndpoint<FindByIdInput, NotificationSubscription>;
    findMany: ApiEndpoint<NotificationSubscriptionSearchInput, NotificationSubscriptionSearchResult>;
    createOne: ApiEndpoint<NotificationSubscriptionCreateInput, NotificationSubscription>;
    updateOne: ApiEndpoint<NotificationSubscriptionUpdateInput, NotificationSubscription>;
}

export const notificationSubscription: EndpointsNotificationSubscription = createStandardCrudEndpoints({
    objectType: "NotificationSubscription",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
        },
        createOne: {
            rateLimit: RateLimitPresets.STRICT,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
