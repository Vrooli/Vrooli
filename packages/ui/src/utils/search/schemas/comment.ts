import { CommentSortBy, FormSchema, endpointsComment } from "@local/shared";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";

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
