import { type Schedule } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const schedule: ApiPartial<Schedule> = {
    common: {
        id: true,
        publicId: true,
        createdAt: true,
        updatedAt: true,
        startTime: true,
        endTime: true,
        timezone: true,
        exceptions: async () => rel((await import("./scheduleException.js")).scheduleException, "list", { omit: "schedule" }),
        recurrences: async () => rel((await import("./scheduleRecurrence.js")).scheduleRecurrence, "list", { omit: "schedule" }),
    },
    full: {
        meetings: async () => rel((await import("./meeting.js")).meeting, "full", { omit: "schedule" }),
        runs: async () => rel((await import("./run.js")).run, "full", { omit: "schedule" }),
    },
    list: {
        meetings: async () => rel((await import("./meeting.js")).meeting, "list", { omit: "schedule" }),
        runs: async () => rel((await import("./run.js")).run, "list", { omit: "schedule" }),
    },
};
