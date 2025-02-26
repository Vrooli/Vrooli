import { Schedule } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const schedule: ApiPartial<Schedule> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        startTime: true,
        endTime: true,
        timezone: true,
        exceptions: async () => rel((await import("./scheduleException.js")).scheduleException, "list", { omit: "schedule" }),
        recurrences: async () => rel((await import("./scheduleRecurrence.js")).scheduleRecurrence, "list", { omit: "schedule" }),
    },
    full: {
        labels: async () => rel((await import("./label.js")).label, "full"),
        focusModes: async () => rel((await import("./focusMode.js")).focusMode, "full", { omit: "schedule" }),
        meetings: async () => rel((await import("./meeting.js")).meeting, "full", { omit: "schedule" }),
        runProjects: async () => rel((await import("./runProject.js")).runProject, "full", { omit: "schedule" }),
        runRoutines: async () => rel((await import("./runRoutine.js")).runRoutine, "full", { omit: "schedule" }),
    },
    list: {
        labels: async () => rel((await import("./label.js")).label, "list"),
        focusModes: async () => rel((await import("./focusMode.js")).focusMode, "list", { omit: "schedule" }),
        meetings: async () => rel((await import("./meeting.js")).meeting, "list", { omit: "schedule" }),
        runProjects: async () => rel((await import("./runProject.js")).runProject, "list", { omit: "schedule" }),
        runRoutines: async () => rel((await import("./runRoutine.js")).runRoutine, "list", { omit: "schedule" }),
    },
};
