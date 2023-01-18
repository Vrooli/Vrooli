import { notificationSubscriptionFields as fullFields, listNotificationSubscriptionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const notificationSubscriptionEndpoint = {
    findOne: toQuery('notificationSubscription', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('notificationSubscriptions', 'NotificationSubscriptionSearchInput', toSearch(listFields)),
    create: toMutation('notificationSubscriptionCreate', 'NotificationSubscriptionCreateInput', fullFields[1]),
    update: toMutation('notificationSubscriptionUpdate', 'NotificationSubscriptionUpdateInput', fullFields[1])
}