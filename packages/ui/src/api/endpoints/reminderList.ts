import { reminderListPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const reminderListEndpoint = {
    findOne: toQuery('reminderList', 'FindByIdInput', reminderListPartial, 'full'),
    findMany: toQuery('reminderLists', 'ReminderListSearchInput', ...toSearch(reminderListPartial)),
    create: toMutation('reminderListCreate', 'ReminderListCreateInput', reminderListPartial, 'full'),
    update: toMutation('reminderListUpdate', 'ReminderListUpdateInput', reminderListPartial, 'full')
}