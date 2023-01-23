import { UserSchedule } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

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
        filters: () => relPartial(require('./userScheduleFilter').userScheduleFilterPartial, 'full'),
        labels: () => relPartial(require('./label').labelPartial, 'full'),
        reminderList: () => relPartial(require('./reminderList').reminderListPartial, 'full', { omit: 'userSchedule' }),
    },
    list: {
        labels: () => relPartial(require('./label').labelPartial, 'list'),
    }
}