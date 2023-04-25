import { BookmarkListSortBy } from "@local/shared";
import { bookmarkListFindMany } from "api/generated/endpoints/bookmarkList_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const bookmarkListSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchBookmarkList'),
    containers: [],
    fields: []
})

export const bookmarkListSearchParams = () => toParams(bookmarkListSearchSchema(), bookmarkListFindMany, BookmarkListSortBy, BookmarkListSortBy.LabelAsc);