import { Bookmark } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const bookmark: ApiPartial<Bookmark> = {
    common: {
        id: true,
        list: async () => rel((await import("./bookmarkList.js")).bookmarkList, "common"),
        to: {
            __union: {
                Api: async () => rel((await import("./api.js")).api, "list"),
                Code: async () => rel((await import("./code.js")).code, "list"),
                Comment: async () => rel((await import("./comment.js")).comment, "common"),
                Issue: async () => rel((await import("./issue.js")).issue, "nav"),
                Note: async () => rel((await import("./note.js")).note, "list"),
                Project: async () => rel((await import("./project.js")).project, "list"),
                Routine: async () => rel((await import("./resource.js")).routine, "list"),
                Standard: async () => rel((await import("./standard.js")).standard, "list"),
                Tag: async () => rel((await import("./tag.js")).tag, "list"),
                Team: async () => rel((await import("./team.js")).team, "list"),
                User: async () => rel((await import("./user.js")).user, "list"),
            },
        },
    },
    full: {
        list: async () => rel((await import("./bookmarkList.js")).bookmarkList, "common", { omit: "bookmarks" }),
    },
};
