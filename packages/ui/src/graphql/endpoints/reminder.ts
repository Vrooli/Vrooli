import { reminderFields as fullFields, listReminderFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reminderEndpoint = {
    findOne: toQuery('reminder', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('reminders', 'ReminderSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('reminderCreate', 'ReminderCreateInput', [fullFields], `...fullFields`),
    update: toMutation('reminderUpdate', 'ReminderUpdateInput', [fullFields], `...fullFields`)
}