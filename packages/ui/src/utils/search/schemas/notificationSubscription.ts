import { NotificationSubscriptionSortBy } from "@local/shared";
import { notificationSubscriptionFindMany } from "api/generated/endpoints/notificationSubscription_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const notificationSubscriptionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchNotificationSubscription'),
    containers: [], //TODO
    fields: [], //TODO
})

export const notificationSubscriptionSearchParams = () => toParams(notificationSubscriptionSearchSchema(), notificationSubscriptionFindMany, NotificationSubscriptionSortBy, NotificationSubscriptionSortBy.DateCreatedDesc);