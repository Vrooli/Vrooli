import { type ScheduleException } from "@local/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const scheduleException: ApiPartial<ScheduleException> = {
    common: {
        id: true,
        originalStartTime: true,
        newStartTime: true,
        newEndTime: true,
    },
    full: {
        schedule: async () => rel((await import("./schedule.js")).schedule, "full", { omit: "exceptions" }),
    },
    list: {
        schedule: async () => rel((await import("./schedule.js")).schedule, "list", { omit: "exceptions" }),
    },
};
