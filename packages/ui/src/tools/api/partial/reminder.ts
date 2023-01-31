import { Reminder } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const reminderPartial: GqlPartial<Reminder> = {
    __typename: 'Reminder',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        description: true,
        dueDate: true,
        completed: true,
        index: true,
        reminderItems: async () => relPartial((await import('./reminderItem')).reminderItemPartial, 'full', { omit: 'reminder' }),
        reminderList: async () => relPartial((await import('./reminderList')).reminderListPartial, 'nav', { omit: 'reminders' }),
    },
}