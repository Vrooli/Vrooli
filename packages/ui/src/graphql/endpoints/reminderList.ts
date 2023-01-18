import { reminderListFields as fullFields, listReminderListFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reminderListEndpoint = {
    findOne: toQuery('reminderList', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('reminderLists', 'ReminderListSearchInput', toSearch(listFields)),
    create: toMutation('reminderListCreate', 'ReminderListCreateInput', fullFields[1]),
    update: toMutation('reminderListUpdate', 'ReminderListUpdateInput', fullFields[1])
}