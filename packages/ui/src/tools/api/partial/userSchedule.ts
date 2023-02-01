import { UserSchedule } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const userSchedule: GqlPartial<UserSchedule> = {
    __typename: 'UserSchedule',
    common: {
        id: true,
        name: true,
        description: true,
        timeZone: true,
        eventStart: true,
        eventEnd: true,
        recurring: true,
        recurrStart: true,
        recurrEnd: true,
    },
    full: {
        __define: {
            0: async () => rel((await import('./label')).label, 'full'),
        },
        filters: async () => rel((await import('./userScheduleFilter')).userScheduleFilter, 'full'),
        labels: { __use: 0 },
        reminderList: async () => rel((await import('./reminderList')).reminderList, 'full', { omit: 'userSchedule' }),
    },
    list: {
        __define: {
            0: async () => rel((await import('./label')).label, 'list'),
        },
        labels: { __use: 0 },
    }
}