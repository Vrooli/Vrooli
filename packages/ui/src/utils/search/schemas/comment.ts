import { CommentSortBy, endpointGetComments } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, votesContainer, votesFields } from "./common";

export const commentSearchSchema = (): FormSchema => ({
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

export const commentSearchParams = () => toParams(commentSearchSchema(), endpointGetComments, CommentSortBy, CommentSortBy.ScoreDesc);
