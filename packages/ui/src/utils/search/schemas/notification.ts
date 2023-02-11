import { NotificationSortBy } from "@shared/consts";
import { notificationFindMany } from "api/generated/endpoints/notification";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const notificationSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchNotifications', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const notificationSearchParams = (lng: string) => toParams(notificationSearchSchema(lng), notificationFindMany, NotificationSortBy, NotificationSortBy.DateCreatedDesc);