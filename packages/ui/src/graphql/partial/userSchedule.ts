import { UserSchedule } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { labelPartial } from "./label";
import { reminderListPartial } from "./reminderList";
import { userScheduleFilterPartial } from "./userScheduleFilter";

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
        filters: () => relPartial(userScheduleFilterPartial, 'full'),
        labels: () => relPartial(labelPartial, 'full'),
        reminderList: () => relPartial(reminderListPartial, 'full', { omit: 'userSchedule' }),
    },
    list: {
        labels: () => relPartial(labelPartial, 'list'),
    }
}