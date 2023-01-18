import { notificationFields, notificationSettingsFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const notificationEndpoint = {
    findOne: toQuery('notification', 'FindByIdInput', notificationFields[1]),
    findMany: toQuery('notifications', 'NotificationSearchInput', toSearch(notificationFields)),
    markAsRead: toMutation('notificationMarkAsRead', 'FindByIdInput', `{ success }`),
    update: toMutation('notificationMarkAllAsRead', null, `{ count }`),
    settingsUpdate: toMutation('notificationSettingsUpdate', 'NotificationSettingsUpdateInput', notificationSettingsFields[1]),
}