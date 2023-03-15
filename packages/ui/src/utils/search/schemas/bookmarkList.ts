import { BookmarkSortBy } from "@shared/consts";
import { bookmarkFindMany } from "api/generated/endpoints/bookmark_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const bookmarkListSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchBookmarkList'),
    containers: [],
    fields: []
})

export const bookmarkListSearchParams = () => toParams(bookmarkListSearchSchema(), bookmarkFindMany, BookmarkSortBy, BookmarkSortBy.DateUpdatedDesc);