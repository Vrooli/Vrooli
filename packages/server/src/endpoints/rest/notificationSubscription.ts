import { notificationSubscription_create, notificationSubscription_findMany, notificationSubscription_findOne, notificationSubscription_update } from "@local/shared";
import { NotificationSubscriptionEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const NotificationSubscriptionRest = setupRoutes({
    "/notificationSubscription/:id": {
        get: [NotificationSubscriptionEndpoints.Query.notificationSubscription, notificationSubscription_findOne],
        put: [NotificationSubscriptionEndpoints.Mutation.notificationSubscriptionUpdate, notificationSubscription_update],
    },
    "/notificationSubscriptions": {
        get: [NotificationSubscriptionEndpoints.Query.notificationSubscriptions, notificationSubscription_findMany],
    },
    "/notificationSubscription": {
        post: [NotificationSubscriptionEndpoints.Mutation.notificationSubscriptionCreate, notificationSubscription_create],
    },
});
