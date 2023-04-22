import { BookmarkListSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { bookmarkListFindMany } from "../../../api/generated/endpoints/bookmarkList_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const bookmarkListSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchBookmarkList"),
    containers: [],
    fields: [],
});

export const bookmarkListSearchParams = () => toParams(bookmarkListSearchSchema(), bookmarkListFindMany, BookmarkListSortBy, BookmarkListSortBy.LabelAsc);
