import { ScheduleException } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const scheduleException: GqlPartial<ScheduleException> = {
    __typename: "ScheduleException",
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
