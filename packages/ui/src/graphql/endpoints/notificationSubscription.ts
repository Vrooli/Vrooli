import { notificationSubscriptionPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const notificationSubscriptionEndpoint = {
    findOne: toQuery('notificationSubscription', 'FindByIdInput', notificationSubscriptionPartial, 'full'),
    findMany: toQuery('notificationSubscriptions', 'NotificationSubscriptionSearchInput', ...toSearch(notificationSubscriptionPartial)),
    create: toMutation('notificationSubscriptionCreate', 'NotificationSubscriptionCreateInput', notificationSubscriptionPartial, 'full'),
    update: toMutation('notificationSubscriptionUpdate', 'NotificationSubscriptionUpdateInput', notificationSubscriptionPartial, 'full')
}