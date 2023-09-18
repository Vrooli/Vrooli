import { ReminderListModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ReminderListFormat: Formatter<ReminderListModelLogic> = {
    gqlRelMap: {
        __typename: "ReminderList",
        focusMode: "FocusMode",
        reminders: "Reminder",
    },
    prismaRelMap: {
        __typename: "ReminderList",
        focusMode: "FocusMode",
        reminders: "Reminder",
    },
    countFields: {},
};
