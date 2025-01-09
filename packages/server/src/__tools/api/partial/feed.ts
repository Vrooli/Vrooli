import { HomeResult, Popular } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const homeResult: ApiPartial<HomeResult> = {
    list: {
        recommended: async () => rel((await import("./resource")).resource, "list"),
        reminders: async () => rel((await import("./reminder")).reminder, "full"),
        resources: async () => rel((await import("./resource")).resource, "list"),
        schedules: async () => rel((await import("./schedule")).schedule, "list"),
    },
};

export const popular: ApiPartial<Popular> = {
    list: {
        __union: {
            Api: async () => rel((await import("./api")).api, "list"),
            Code: async () => rel((await import("./code")).code, "list"),
            Note: async () => rel((await import("./note")).note, "list"),
            Project: async () => rel((await import("./project")).project, "list"),
            Question: async () => rel((await import("./question")).question, "list"),
            Routine: async () => rel((await import("./routine")).routine, "list"),
            Standard: async () => rel((await import("./standard")).standard, "list"),
            Team: async () => rel((await import("./team")).team, "list"),
            User: async () => rel((await import("./user")).user, "list"),
        },
    },
};
