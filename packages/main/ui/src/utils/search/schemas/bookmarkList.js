import { BookmarkListSortBy } from "@local/consts";
import { bookmarkListFindMany } from "../../../api/generated/endpoints/bookmarkList_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const bookmarkListSearchSchema = () => ({
    formLayout: searchFormLayout("SearchBookmarkList"),
    containers: [],
    fields: [],
});
export const bookmarkListSearchParams = () => toParams(bookmarkListSearchSchema(), bookmarkListFindMany, BookmarkListSortBy, BookmarkListSortBy.LabelAsc);
//# sourceMappingURL=bookmarkList.js.map