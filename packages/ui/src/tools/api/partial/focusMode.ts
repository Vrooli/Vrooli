import { FocusMode } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const focusMode: GqlPartial<FocusMode> = {
    __typename: "FocusMode",
    common: {
        id: true,
        name: true,
        description: true,
    },
    full: {
        __define: {
            0: async () => rel((await import("./schedule")).schedule, "common"),
        },
        filters: async () => rel((await import("./focusModeFilter")).focusModeFilter, "full"),
        labels: {
            id: true,
            color: true,
            label: true,
        },
        reminderList: async () => rel((await import("./reminderList")).reminderList, "full", { omit: "focusMode" }),
        resourceList: async () => rel((await import("./resourceList")).resourceList, "full", { omit: "focusMode" }),
        schedule: { __use: 0 },
    },
    list: {
        __define: {
            0: async () => rel((await import("./schedule")).schedule, "common"),
        },
        labels: {
            id: true,
            color: true,
            label: true,
        },
        reminderList: async () => rel((await import("./reminderList")).reminderList, "nav", { omit: "focusMode" }),
        resourceList: async () => rel((await import("./resourceList")).resourceList, "nav", { omit: "focusMode" }),
        schedule: { __use: 0 },
    },
};
