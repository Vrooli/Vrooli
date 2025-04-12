import { View } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const view: ApiPartial<View> = {
    list: {
        to: {
            __union: {
                Api: async () => rel((await import("./api.js")).api, "list"),
                Code: async () => rel((await import("./code.js")).code, "list"),
                Issue: async () => rel((await import("./issue.js")).issue, "list"),
                Note: async () => rel((await import("./note.js")).note, "list"),
                Post: async () => rel((await import("./post.js")).post, "list"),
                Project: async () => rel((await import("./project.js")).project, "list"),
                Question: async () => rel((await import("./question.js")).question, "list"),
                Routine: async () => rel((await import("./routine.js")).routine, "list"),
                Standard: async () => rel((await import("./standard.js")).standard, "list"),
                Team: async () => rel((await import("./team.js")).team, "list"),
                User: async () => rel((await import("./user.js")).user, "list"),
            },
        },
    },
};
