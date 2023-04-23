import { NotificationSubscriptionSortBy } from "@local/consts";
import { notificationSubscriptionFindMany } from "../../../api/generated/endpoints/notificationSubscription_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const notificationSubscriptionSearchSchema = () => ({
    formLayout: searchFormLayout("SearchNotificationSubscription"),
    containers: [],
    fields: [],
});
export const notificationSubscriptionSearchParams = () => toParams(notificationSubscriptionSearchSchema(), notificationSubscriptionFindMany, NotificationSubscriptionSortBy, NotificationSubscriptionSortBy.DateCreatedDesc);
//# sourceMappingURL=notificationSubscription.js.map