import { endpointsNotification, FormSchema, NotificationSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function notificationSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchNotification"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function notificationSearchParams() {
    return toParams(notificationSearchSchema(), endpointsNotification, NotificationSortBy, NotificationSortBy.DateCreatedDesc);
}
