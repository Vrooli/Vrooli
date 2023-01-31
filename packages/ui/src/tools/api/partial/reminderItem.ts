import { ReminderItem } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const reminderItemPartial: GqlPartial<ReminderItem> = {
    __typename: 'ReminderItem',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        description: true,
        dueDate: true,
        completed: true,
        index: true,
        reminder: () => relPartial(require('./reminder').reminderPartial, 'nav', { omit: 'reminderItems' }),
    },
}