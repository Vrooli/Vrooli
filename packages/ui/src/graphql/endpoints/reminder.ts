import { reminderPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const reminderEndpoint = {
    findOne: toQuery('reminder', 'FindByIdInput', reminderPartial, 'full'),
    findMany: toQuery('reminders', 'ReminderSearchInput', ...toSearch(reminderPartial)),
    create: toMutation('reminderCreate', 'ReminderCreateInput', reminderPartial, 'full'),
    update: toMutation('reminderUpdate', 'ReminderUpdateInput', reminderPartial, 'full')
}