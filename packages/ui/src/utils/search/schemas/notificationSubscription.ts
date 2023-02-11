import { NotificationSubscriptionSortBy } from "@shared/consts";
import { notificationSubscriptionFindMany } from "api/generated/endpoints/notificationSubscription";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const notificationSubscriptionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchSubscriptions', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const notificationSubscriptionSearchParams = (lng: string) => toParams(notificationSubscriptionSearchSchema(lng), notificationSubscriptionFindMany, NotificationSubscriptionSortBy, NotificationSubscriptionSortBy.DateCreatedDesc);