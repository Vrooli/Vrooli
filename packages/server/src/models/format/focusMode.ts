import { FocusModeModelLogic } from "../base/types";
import { Formatter } from "../types";

export const FocusModeFormat: Formatter<FocusModeModelLogic> = {
    gqlRelMap: {
        __typename: "FocusMode",
        filters: "FocusModeFilter",
        labels: "Label",
        reminderList: "ReminderList",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "FocusMode",
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
