import { HomeResult, Popular } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const homeResult: ApiPartial<HomeResult> = {
    list: {
        reminders: async () => rel((await import("./reminder.js")).reminder, "full"),
        schedules: async () => rel((await import("./schedule.js")).schedule, "list"),
    },
};

export const popular: ApiPartial<Popular> = {
    list: {
        __union: {
            Api: async () => rel((await import("./api.js")).api, "list"),
            Code: async () => rel((await import("./code.js")).code, "list"),
            Note: async () => rel((await import("./note.js")).note, "list"),
            Project: async () => rel((await import("./project.js")).project, "list"),
            Routine: async () => rel((await import("./resource.js")).routine, "list"),
            Standard: async () => rel((await import("./standard.js")).standard, "list"),
            Team: async () => rel((await import("./team.js")).team, "list"),
            User: async () => rel((await import("./user.js")).user, "list"),
        },
    },
};
