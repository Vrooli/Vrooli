import { ReminderModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ReminderFormat: Formatter<ReminderModelLogic> = {
    gqlRelMap: {
        __typename: "Reminder",
        reminderItems: "ReminderItem",
        reminderList: "ReminderList",
    },
    prismaRelMap: {
        __typename: "Reminder",
        reminderItems: "ReminderItem",
        reminderList: "ReminderList",
    },
    countFields: {},
};
