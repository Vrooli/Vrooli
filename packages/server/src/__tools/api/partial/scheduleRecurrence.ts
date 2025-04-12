import { ScheduleRecurrence } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const scheduleRecurrence: ApiPartial<ScheduleRecurrence> = {
    common: {
        id: true,
        recurrenceType: true,
        interval: true,
        dayOfWeek: true,
        dayOfMonth: true,
        month: true,
        endDate: true,
    },
    full: {
        schedule: async () => rel((await import("./schedule.js")).schedule, "full", { omit: "recurrences" }),
    },
    list: {
        schedule: async () => rel((await import("./schedule.js")).schedule, "list", { omit: "recurrences" }),
    },
};
