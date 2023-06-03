import { NotificationSubscriptionSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const notificationSubscriptionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchNotificationSubscription"),
    containers: [], //TODO
    fields: [], //TODO
});

export const notificationSubscriptionSearchParams = () => toParams(notificationSubscriptionSearchSchema(), "/notificationSubscriptions", NotificationSubscriptionSortBy, NotificationSubscriptionSortBy.DateCreatedDesc);
