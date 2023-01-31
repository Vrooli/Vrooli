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
            0: () => relPartial(require('./label').labelPartial, 'full'),
        },
        filters: () => relPartial(require('./userScheduleFilter').userScheduleFilterPartial, 'full'),
        labels: { __use: 0 },
        reminderList: () => relPartial(require('./reminderList').reminderListPartial, 'full', { omit: 'userSchedule' }),
    },
    list: {
        __define: {
            0: () => relPartial(require('./label').labelPartial, 'list'),
        },
        labels: { __use: 0 },
    }
}