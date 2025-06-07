import { type BookmarkList } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const bookmarkList: ApiPartial<BookmarkList> = {
    common: {
        id: true,
        createdAt: true,
        updatedAt: true,
        label: true,
        bookmarksCount: true,
    },
    list: {
        bookmarks: async () => rel((await import("./bookmark.js")).bookmark, "list", { omit: "list" }),
    },
    full: {
        bookmarks: async () => rel((await import("./bookmark.js")).bookmark, "full", { omit: "list" }),
    },
};
