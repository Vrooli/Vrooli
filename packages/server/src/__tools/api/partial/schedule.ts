import { Schedule } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const schedule: ApiPartial<Schedule> = {
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
        labels: async () => rel((await import("./label")).label, "full"),
        focusModes: async () => rel((await import("./focusMode")).focusMode, "full", { omit: "schedule" }),
        meetings: async () => rel((await import("./meeting")).meeting, "full", { omit: "schedule" }),
        runProjects: async () => rel((await import("./runProject")).runProject, "full", { omit: "schedule" }),
        runRoutines: async () => rel((await import("./runRoutine")).runRoutine, "full", { omit: "schedule" }),
    },
    list: {
        labels: async () => rel((await import("./label")).label, "list"),
        focusModes: async () => rel((await import("./focusMode")).focusMode, "list", { omit: "schedule" }),
        meetings: async () => rel((await import("./meeting")).meeting, "list", { omit: "schedule" }),
        runProjects: async () => rel((await import("./runProject")).runProject, "list", { omit: "schedule" }),
        runRoutines: async () => rel((await import("./runRoutine")).runRoutine, "list", { omit: "schedule" }),
    },
};
