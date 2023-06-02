import { ReminderListModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ReminderList" as const;
export const ReminderListFormat: Formatter<ReminderListModelLogic> = {
    gqlRelMap: {
        __typename,
        focusMode: "FocusMode",
        reminders: "Reminder",
    },
    prismaRelMap: {
        __typename,
        focusMode: "FocusMode",
        reminders: "Reminder",
    },
    countFields: {},
};
