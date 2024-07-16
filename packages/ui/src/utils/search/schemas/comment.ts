import { CommentSortBy, endpointGetComment, endpointGetComments } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";

export const commentSearchSchema = (): FormSchema => ({
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
});

export const commentSearchParams = () => toParams(commentSearchSchema(), endpointGetComments, endpointGetComment, CommentSortBy, CommentSortBy.ScoreDesc);
