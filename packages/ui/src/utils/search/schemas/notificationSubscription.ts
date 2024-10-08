import { endpointGetNotificationSubscription, endpointGetNotificationSubscriptions, FormSchema, NotificationSubscriptionSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const notificationSubscriptionSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchNotificationSubscription"),
    containers: [], //TODO
    elements: [], //TODO
});

export const notificationSubscriptionSearchParams = () => toParams(notificationSubscriptionSearchSchema(), endpointGetNotificationSubscriptions, endpointGetNotificationSubscription, NotificationSubscriptionSortBy, NotificationSubscriptionSortBy.DateCreatedDesc);
