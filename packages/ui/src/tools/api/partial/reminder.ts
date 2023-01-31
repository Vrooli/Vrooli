import { Reminder } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const reminder: GqlPartial<Reminder> = {
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
        reminderItems: async () => rel((await import('./reminderItem')).reminderItem, 'full', { omit: 'reminder' }),
        reminderList: async () => rel((await import('./reminderList')).reminderList, 'nav', { omit: 'reminders' }),
    },
}