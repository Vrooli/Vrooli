import { FocusMode, FocusModeYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        filters: async () => rel((await import("./focusModeFilter")).focusModeFilter, "full"),
        labels: {
            id: true,
            color: true,
            label: true,
        },
        reminderList: async () => rel((await import("./reminderList")).reminderList, "full", { omit: "focusMode" }),
        resourceList: async () => rel((await import("./resourceList")).resourceList, "common", { omit: "focusMode" }),
        schedule: async () => rel((await import("./schedule")).schedule, "common"),
    },
    list: {
        labels: {
            id: true,
            color: true,
            label: true,
        },
        reminderList: async () => rel((await import("./reminderList")).reminderList, "full", { omit: "focusMode" }),
        resourceList: async () => rel((await import("./resourceList")).resourceList, "common", { omit: "focusMode" }),
        schedule: async () => rel((await import("./schedule")).schedule, "common"),
    },
};
