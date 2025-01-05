import { BookmarkList } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const bookmarkList: GqlPartial<BookmarkList> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        label: true,
        bookmarksCount: true,
    },
    list: {
        bookmarks: async () => rel((await import("./bookmark")).bookmark, "list", { omit: "list" }),
    },
    full: {
        bookmarks: async () => rel((await import("./bookmark")).bookmark, "full", { omit: "list" }),
    },
};
