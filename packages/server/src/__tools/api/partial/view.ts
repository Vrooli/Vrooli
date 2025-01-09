import { View } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const view: ApiPartial<View> = {
    list: {
        to: {
            __union: {
                Api: async () => rel((await import("./api")).api, "list"),
                Code: async () => rel((await import("./code")).code, "list"),
                Issue: async () => rel((await import("./issue")).issue, "list"),
                Note: async () => rel((await import("./note")).note, "list"),
                Post: async () => rel((await import("./post")).post, "list"),
                Project: async () => rel((await import("./project")).project, "list"),
                Question: async () => rel((await import("./question")).question, "list"),
                Routine: async () => rel((await import("./routine")).routine, "list"),
                Standard: async () => rel((await import("./standard")).standard, "list"),
                Team: async () => rel((await import("./team")).team, "list"),
                User: async () => rel((await import("./user")).user, "list"),
            },
        },
    },
};
