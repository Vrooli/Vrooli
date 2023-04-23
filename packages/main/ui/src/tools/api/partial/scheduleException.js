import { rel } from "../utils";
export const scheduleException = {
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
//# sourceMappingURL=scheduleException.js.map