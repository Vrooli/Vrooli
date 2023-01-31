import { ReminderList } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const reminderListPartial: GqlPartial<ReminderList> = {
    __typename: 'ReminderList',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        reminders: () => relPartial(require('./reminder').reminderPartial, 'full', { omit: 'reminderList' }),
        userSchedule: () => relPartial(require('./userSchedule').userSchedulePartial, 'list', { omit: 'reminderList' }),
    },
}