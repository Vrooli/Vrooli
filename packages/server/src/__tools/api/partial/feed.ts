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
            Resource: async () => rel((await import("./resource.js")).resource, "list"),
            Team: async () => rel((await import("./team.js")).team, "list"),
            User: async () => rel((await import("./user.js")).user, "list"),
        },
    },
};
