import { ScheduleException } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const scheduleException: ApiPartial<ScheduleException> = {
    common: {
        id: true,
        originalStartTime: true,
        newStartTime: true,
        newEndTime: true,
    },
    full: {
        schedule: async () => rel((await import("./schedule")).schedule, "full", { omit: "exceptions" }),
    },
    list: {
        schedule: async () => rel((await import("./schedule")).schedule, "list", { omit: "exceptions" }),
    },
};
