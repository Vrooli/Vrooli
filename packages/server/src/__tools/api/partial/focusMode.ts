import { FocusMode, FocusModeYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const focusModeYou: ApiPartial<FocusModeYou> = {
    full: {
        canDelete: true,
        canRead: true,
        canUpdate: true,
    },
};

export const focusMode: ApiPartial<FocusMode> = {
    common: {
        id: true,
        name: true,
        description: true,
        you: () => rel(focusModeYou, "full"),
    },
    full: {
        filters: async () => rel((await import("./focusModeFilter.js")).focusModeFilter, "full"),
        labels: {
            id: true,
            color: true,
            label: true,
        },
        reminderList: async () => rel((await import("./reminderList.js")).reminderList, "full", { omit: "focusMode" }),
        resourceList: async () => rel((await import("./resourceList.js")).resourceList, "common", { omit: "focusMode" }),
        schedule: async () => rel((await import("./schedule.js")).schedule, "common"),
    },
    list: {
        labels: {
            id: true,
            color: true,
            label: true,
        },
        reminderList: async () => rel((await import("./reminderList.js")).reminderList, "full", { omit: "focusMode" }),
        resourceList: async () => rel((await import("./resourceList.js")).resourceList, "common", { omit: "focusMode" }),
        schedule: async () => rel((await import("./schedule.js")).schedule, "common"),
    },
};
