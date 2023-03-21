import { ReminderList } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const reminderList: GqlPartial<ReminderList> = {
    __typename: 'ReminderList',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        focusMode: async () => rel((await import('./focusMode')).focusMode, 'list', { omit: 'reminderList' }),
        reminders: async () => rel((await import('./reminder')).reminder, 'full', { omit: 'reminderList' }),
    },
}