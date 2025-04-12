import { BookmarkList } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const bookmarkList: ApiPartial<BookmarkList> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
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
