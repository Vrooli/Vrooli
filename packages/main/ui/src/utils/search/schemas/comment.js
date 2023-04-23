import { CommentSortBy } from "@local/consts";
import { commentFindMany } from "../../../api/generated/endpoints/comment_findMany";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";
export const commentSearchSchema = () => ({
    formLayout: searchFormLayout("SearchComment"),
    containers: [
        votesContainer(),
        bookmarksContainer(),
        languagesContainer(),
    ],
    fields: [
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesFields(),
    ],
});
export const commentSearchParams = () => toParams(commentSearchSchema(), commentFindMany, CommentSortBy, CommentSortBy.ScoreDesc);
//# sourceMappingURL=comment.js.map