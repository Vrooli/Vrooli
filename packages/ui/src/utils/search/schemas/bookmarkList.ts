import { BookmarkListSortBy, endpointGetBookmarkLists } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const bookmarkListSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchBookmarkList"),
    containers: [],
    fields: [],
});

export const bookmarkListSearchParams = () => toParams(bookmarkListSearchSchema(), endpointGetBookmarkLists, BookmarkListSortBy, BookmarkListSortBy.LabelAsc);
