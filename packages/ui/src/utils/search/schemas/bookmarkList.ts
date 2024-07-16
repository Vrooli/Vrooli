import { BookmarkListSortBy, endpointGetBookmarkList, endpointGetBookmarkLists } from "@local/shared";
import { FormSchema } from "forms/types";
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
    return toParams(bookmarkListSearchSchema(), endpointGetBookmarkLists, endpointGetBookmarkList, BookmarkListSortBy, BookmarkListSortBy.LabelAsc);
}

