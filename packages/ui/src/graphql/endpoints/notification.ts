import { notificationFields, notificationSettingsFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const notificationEndpoint = {
    findOne: toQuery('notification', 'FindByIdInput', [notificationFields], `...notificationFields`),
    findMany: toQuery('notifications', 'NotificationSearchInput', [notificationFields], toSearch(notificationFields)),
    markAsRead: toMutation('notificationMarkAsRead', 'FindByIdInput', [], `success`),
    update: toMutation('notificationMarkAllAsRead', null, [], `count`),
    settingsUpdate: toMutation('notificationSettingsUpdate', 'NotificationSettingsUpdateInput', [notificationSettingsFields], `...notificationSettingsFields`),
}