import { PostSortBy } from "@local/consts";
import { postFindMany } from "../../../api/generated/endpoints/post_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const postSearchSchema = () => ({
    formLayout: searchFormLayout("SearchPost"),
    containers: [],
    fields: [],
});
export const postSearchParams = () => toParams(postSearchSchema(), postFindMany, PostSortBy, PostSortBy.DateCreatedDesc);
//# sourceMappingURL=post.js.map