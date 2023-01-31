import { ReminderList } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const reminderList: GqlPartial<ReminderList> = {
    __typename: 'ReminderList',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        reminders: async () => rel((await import('./reminder')).reminder, 'full', { omit: 'reminderList' }),
        userSchedule: async () => rel((await import('./userSchedule')).userSchedule, 'list', { omit: 'reminderList' }),
    },
}