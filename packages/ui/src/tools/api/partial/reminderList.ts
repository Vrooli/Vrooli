import { ReminderList } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const reminderListPartial: GqlPartial<ReminderList> = {
    __typename: 'ReminderList',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        reminders: async () => relPartial((await import('./reminder')).reminderPartial, 'full', { omit: 'reminderList' }),
        userSchedule: async () => relPartial((await import('./userSchedule')).userSchedulePartial, 'list', { omit: 'reminderList' }),
    },
}