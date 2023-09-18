import { ReminderItemModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ReminderItemFormat: Formatter<ReminderItemModelLogic> = {
    gqlRelMap: {
        __typename: "ReminderItem",
        reminder: "Reminder",
    },
    prismaRelMap: {
        __typename: "ReminderItem",
        reminder: "Reminder",
    },
    countFields: {},
};
