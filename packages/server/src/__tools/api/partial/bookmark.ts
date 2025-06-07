import { type Bookmark } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const bookmark: ApiPartial<Bookmark> = {
    common: {
        id: true,
        list: async () => rel((await import("./bookmarkList.js")).bookmarkList, "common"),
        to: {
            __union: {
                Comment: async () => rel((await import("./comment.js")).comment, "common"),
                Issue: async () => rel((await import("./issue.js")).issue, "nav"),
                Resource: async () => rel((await import("./resource.js")).resource, "list"),
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
