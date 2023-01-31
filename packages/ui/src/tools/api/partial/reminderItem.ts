import { ReminderItem } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const reminderItem: GqlPartial<ReminderItem> = {
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
        reminder: async () => rel((await import('./reminder')).reminder, 'nav', { omit: 'reminderItems' }),
    },
}