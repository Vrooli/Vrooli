import { BookmarkListSortBy, FormSchema, endpointsBookmarkList } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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

