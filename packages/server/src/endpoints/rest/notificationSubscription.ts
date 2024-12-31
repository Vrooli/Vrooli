import { endpointsNotificationSubscription } from "@local/shared";
import { notificationSubscription_create, notificationSubscription_findMany, notificationSubscription_findOne, notificationSubscription_update } from "../generated";
import { NotificationSubscriptionEndpoints } from "../logic/notificationSubscription";
import { setupRoutes } from "./base";

export const NotificationSubscriptionRest = setupRoutes([
    [endpointsNotificationSubscription.findOne, NotificationSubscriptionEndpoints.Query.notificationSubscription, notificationSubscription_findOne],
    [endpointsNotificationSubscription.findMany, NotificationSubscriptionEndpoints.Query.notificationSubscriptions, notificationSubscription_findMany],
    [endpointsNotificationSubscription.createOne, NotificationSubscriptionEndpoints.Mutation.notificationSubscriptionCreate, notificationSubscription_create],
    [endpointsNotificationSubscription.updateOne, NotificationSubscriptionEndpoints.Mutation.notificationSubscriptionUpdate, notificationSubscription_update],
]);
