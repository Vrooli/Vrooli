import { endpointsNotificationSubscription, FormSchema, NotificationSubscriptionSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function notificationSubscriptionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchNotificationSubscription"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function notificationSubscriptionSearchParams() {
    return toParams(notificationSubscriptionSearchSchema(), endpointsNotificationSubscription, NotificationSubscriptionSortBy, NotificationSubscriptionSortBy.DateCreatedDesc);
}
