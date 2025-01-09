import { ScheduleRecurrence } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        schedule: async () => rel((await import("./schedule")).schedule, "full", { omit: "recurrences" }),
    },
    list: {
        schedule: async () => rel((await import("./schedule")).schedule, "list", { omit: "recurrences" }),
    },
};
