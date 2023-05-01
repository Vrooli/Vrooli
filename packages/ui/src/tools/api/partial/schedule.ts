import { Schedule } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const schedule: GqlPartial<Schedule> = {
    __typename: "Schedule",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        startTime: true,
        endTime: true,
        timezone: true,
        exceptions: async () => rel((await import("./scheduleException")).scheduleException, "list", { omit: "schedule" }),
        recurrences: async () => rel((await import("./scheduleRecurrence")).scheduleRecurrence, "list", { omit: "schedule" }),
    },
    full: {
        __define: {
            0: async () => rel((await import("./label")).label, "full"),
        },
        labels: { __use: 0 },
    },
    list: {
        __define: {
            0: async () => rel((await import("./label")).label, "list"),
        },
        labels: { __use: 0 },
    },
};
