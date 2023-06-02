import { ReminderItemModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ReminderItem" as const;
export const ReminderItemFormat: Formatter<ReminderItemModelLogic> = {
    gqlRelMap: {
        __typename,
        reminder: "Reminder",
    },
    prismaRelMap: {
        __typename,
        reminder: "Reminder",
    },
    countFields: {},
};
