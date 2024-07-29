import { endpointGetNotification, endpointGetNotifications, FormSchema, NotificationSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const notificationSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchNotification"),
    containers: [], //TODO
    elements: [], //TODO
});

export const notificationSearchParams = () => toParams(notificationSearchSchema(), endpointGetNotifications, endpointGetNotification, NotificationSortBy, NotificationSortBy.DateCreatedDesc);
