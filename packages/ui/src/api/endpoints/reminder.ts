import { reminderPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const reminderEndpoint = {
    findOne: toQuery('reminder', 'FindByIdInput', reminderPartial, 'full'),
    findMany: toQuery('reminders', 'ReminderSearchInput', ...toSearch(reminderPartial)),
    create: toMutation('reminderCreate', 'ReminderCreateInput', reminderPartial, 'full'),
    update: toMutation('reminderUpdate', 'ReminderUpdateInput', reminderPartial, 'full')
}