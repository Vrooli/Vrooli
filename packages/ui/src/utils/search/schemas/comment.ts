import { CommentSortBy, endpointsComment, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common.js";

export function commentSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchComment"),
        containers: [
            votesContainer(),
            bookmarksContainer(),
            languagesContainer(),
        ],
        elements: [
            ...votesFields(),
            ...bookmarksFields(),
            ...languagesFields(),
        ],
    };
}

export function commentSearchParams() {
    return toParams(commentSearchSchema(), endpointsComment, CommentSortBy, CommentSortBy.ScoreDesc);
}
