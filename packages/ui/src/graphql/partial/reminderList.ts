import { ReminderList } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { reminderPartial } from "./reminder";
import { userSchedulePartial } from "./userSchedule";

export const remindeListPartial: GqlPartial<ReminderList> = {
    __typename: 'ReminderList',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        reminders: () => relPartial(reminderPartial, 'full', { omit: 'reminderList' }),
        userSchedule: () => relPartial(userSchedulePartial, 'nav', { omit: 'reminderList' }),
    },
}