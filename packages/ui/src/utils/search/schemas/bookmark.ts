import { BookmarkSortBy } from "@shared/consts";
import { bookmarkFindMany } from "api/generated/endpoints/bookmark";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const bookmarkSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchBookmark', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const bookmarkSearchParams = (lng: string) => toParams(bookmarkSearchSchema(lng), bookmarkFindMany, BookmarkSortBy, BookmarkSortBy.DateUpdatedDesc);