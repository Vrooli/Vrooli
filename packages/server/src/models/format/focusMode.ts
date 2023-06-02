import { FocusModeModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "FocusMode" as const;
export const FocusModeFormat: Formatter<FocusModeModelLogic> = {
    gqlRelMap: {
        __typename,
        filters: "FocusModeFilter",
        labels: "Label",
        reminderList: "ReminderList",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename,
        reminderList: "ReminderList",
        resourceList: "ResourceList",
        user: "User",
        labels: "Label",
        filters: "FocusModeFilter",
        schedule: "Schedule",
    },
    countFields: {},
    joinMap: { labels: "label" },
};
