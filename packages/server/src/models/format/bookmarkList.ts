import { BookmarkListModelLogic } from "../base/types";
import { Formatter } from "../types";

export const BookmarkListFormat: Formatter<BookmarkListModelLogic> = {
    gqlRelMap: {
        __typename: "BookmarkList",
        bookmarks: "Bookmark",
    },
    prismaRelMap: {
        __typename: "BookmarkList",
        bookmarks: "Bookmark",
    },
    countFields: {
        bookmarksCount: true,
    },
};
