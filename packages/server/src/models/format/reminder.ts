import { ReminderModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Reminder" as const;
export const ReminderFormat: Formatter<ReminderModelLogic> = {
    gqlRelMap: {
        __typename,
        reminderItems: "ReminderItem",
        reminderList: "ReminderList",
    },
    prismaRelMap: {
        __typename,
        reminderItems: "ReminderItem",
        reminderList: "ReminderList",
    },
    countFields: {},
};
