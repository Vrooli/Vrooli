import { NotificationSortBy } from "@shared/consts";
import { notificationFindMany } from "api/generated/endpoints/notification";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const notificationSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchNotification'),
    containers: [], //TODO
    fields: [], //TODO
})

export const notificationSearchParams = () => toParams(notificationSearchSchema(), notificationFindMany, NotificationSortBy, NotificationSortBy.DateCreatedDesc);