import { UserSchedule } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const userSchedulePartial: GqlPartial<UserSchedule> = {
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
            0: async () => relPartial((await import('./label')).labelPartial, 'full'),
        },
        filters: async () => relPartial((await import('./userScheduleFilter')).userScheduleFilterPartial, 'full'),
        labels: { __use: 0 },
        reminderList: async () => relPartial((await import('./reminderList')).reminderListPartial, 'full', { omit: 'userSchedule' }),
    },
    list: {
        __define: {
            0: async () => relPartial((await import('./label')).labelPartial, 'list'),
        },
        labels: { __use: 0 },
    }
}