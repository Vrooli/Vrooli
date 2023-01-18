import { reminderFields as fullFields, listReminderFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reminderEndpoint = {
    findOne: toQuery('reminder', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('reminders', 'ReminderSearchInput', toSearch(listFields)),
    create: toMutation('reminderCreate', 'ReminderCreateInput', fullFields[1]),
    update: toMutation('reminderUpdate', 'ReminderUpdateInput', fullFields[1])
}