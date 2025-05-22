import { BookmarkListSortBy, type FormSchema, endpointsBookmarkList } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function bookmarkListSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchBookmarkList"),
        containers: [],
        elements: [],
    };
}

export function bookmarkListSearchParams() {
    return toParams(bookmarkListSearchSchema(), endpointsBookmarkList, BookmarkListSortBy, BookmarkListSortBy.LabelAsc);
}

