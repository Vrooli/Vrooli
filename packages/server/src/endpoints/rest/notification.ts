import { notification_findMany, notification_findOne, notification_markAllAsRead, notification_markAsRead, notification_settings, notification_settingsUpdate } from "../generated";
import { NotificationEndpoints } from "../logic/notification";
import { setupRoutes } from "./base";

export const NotificationRest = setupRoutes({
    "/notification/:id": {
        get: [NotificationEndpoints.Query.notification, notification_findOne],
        put: [NotificationEndpoints.Mutation.notificationMarkAsRead, notification_markAsRead],
    },
    "/notifications": {
        get: [NotificationEndpoints.Query.notifications, notification_findMany],
    },
    "/notifications/markAllAsRead": {
        put: [NotificationEndpoints.Mutation.notificationMarkAllAsRead, notification_markAllAsRead],
    },
    "/notificationSettings": {
        get: [NotificationEndpoints.Query.notificationSettings, notification_settings],
        put: [NotificationEndpoints.Mutation.notificationSettingsUpdate, notification_settingsUpdate],
    },
});
