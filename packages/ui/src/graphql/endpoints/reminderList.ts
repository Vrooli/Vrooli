import { reminderListFields as fullFields, listReminderListFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reminderListEndpoint = {
    findOne: toQuery('reminderList', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('reminderLists', 'ReminderListSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('reminderListCreate', 'ReminderListCreateInput', [fullFields], `...fullFields`),
    update: toMutation('reminderListUpdate', 'ReminderListUpdateInput', [fullFields], `...fullFields`)
}