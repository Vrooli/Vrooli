import { TagSortBy } from "@local/consts";
import { tagFindMany } from "../../../api/generated/endpoints/tag_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const tagSearchSchema = () => ({
    formLayout: searchFormLayout("SearchTag"),
    containers: [],
    fields: [],
});
export const tagSearchParams = () => toParams(tagSearchSchema(), tagFindMany, TagSortBy, TagSortBy.BookmarksDesc);
//# sourceMappingURL=tag.js.map