import { notificationSubscriptionFields as fullFields, listNotificationSubscriptionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const notificationSubscriptionEndpoint = {
    findOne: toQuery('notificationSubscription', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('notificationSubscriptions', 'NotificationSubscriptionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('notificationSubscriptionCreate', 'NotificationSubscriptionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('notificationSubscriptionUpdate', 'NotificationSubscriptionUpdateInput', [fullFields], `...fullFields`)
}