import { NotificationSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { notificationFindMany } from "../../../api/generated/endpoints/notification_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const notificationSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchNotification"),
    containers: [], //TODO
    fields: [], //TODO
})

export const notificationSearchParams = () => toParams(notificationSearchSchema(), notificationFindMany, NotificationSortBy, NotificationSortBy.DateCreatedDesc);