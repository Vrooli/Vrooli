import { NotificationSortBy } from "@local/consts";
import { notificationFindMany } from "../../../api/generated/endpoints/notification_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const notificationSearchSchema = () => ({
    formLayout: searchFormLayout("SearchNotification"),
    containers: [],
    fields: [],
});
export const notificationSearchParams = () => toParams(notificationSearchSchema(), notificationFindMany, NotificationSortBy, NotificationSortBy.DateCreatedDesc);
//# sourceMappingURL=notification.js.map