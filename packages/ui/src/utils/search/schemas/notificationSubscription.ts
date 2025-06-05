import { NotificationSubscriptionSortBy, endpointsNotificationSubscription, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
