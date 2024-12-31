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
        focusModes: async () => rel((await import("./focusMode")).focusMode, "full", { omit: "schedule" }),
        meetings: async () => rel((await import("./meeting")).meeting, "full", { omit: "schedule" }),
        runProjects: async () => rel((await import("./runProject")).runProject, "full", { omit: "schedule" }),
        runRoutines: async () => rel((await import("./runRoutine")).runRoutine, "full", { omit: "schedule" }),
    },
    list: {
        __define: {
            0: async () => rel((await import("./label")).label, "list"),
        },
        labels: { __use: 0 },
        focusModes: async () => rel((await import("./focusMode")).focusMode, "list", { omit: "schedule" }),
        meetings: async () => rel((await import("./meeting")).meeting, "list", { omit: "schedule" }),
        runProjects: async () => rel((await import("./runProject")).runProject, "list", { omit: "schedule" }),
        runRoutines: async () => rel((await import("./runRoutine")).runRoutine, "list", { omit: "schedule" }),
    },
};
