import { BookmarkList } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const bookmarkList: GqlPartial<BookmarkList> = {
    __typename: 'BookmarkList',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        label: true,
        bookmarksCount: true,
    },
    list: {
        __define: {
            0: async () => rel((await import('./bookmark')).bookmark, 'list', { omit: 'list' }),
        },
        bookmarks: { __use: 0 },
    },
    full: {
        __define: {
            0: async () => rel((await import('./bookmark')).bookmark, 'full', { omit: 'list' }),
        },
        bookmarks: { __use: 0 },
    }
}