import { countPartial, notificationPartial, notificationSettingsPartial, successPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const notificationEndpoint = {
    findOne: toQuery('notification', 'FindByIdInput', notificationPartial, 'full'),
    findMany: toQuery('notifications', 'NotificationSearchInput', ...toSearch(notificationPartial)),
    markAsRead: toMutation('notificationMarkAsRead', 'FindByIdInput', successPartial, 'full'),
    update: toMutation('notificationMarkAllAsRead', null, countPartial, 'full'),
    settingsUpdate: toMutation('notificationSettingsUpdate', 'NotificationSettingsUpdateInput', notificationSettingsPartial, 'full'),
}