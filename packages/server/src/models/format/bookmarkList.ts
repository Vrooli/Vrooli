import { BookmarkListModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "BookmarkList" as const;
export const BookmarkListFormat: Formatter<BookmarkListModelLogic> = {
    gqlRelMap: {
        __typename,
        bookmarks: "Bookmark",
    },
    prismaRelMap: {
        __typename,
        bookmarks: "Bookmark",
    },
    countFields: {
        bookmarksCount: true,
    },
};
