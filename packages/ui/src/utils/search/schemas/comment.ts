import { CommentSortBy } from "@shared/consts";
import { commentFindMany } from "api/generated/endpoints/comment";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, bookmarksContainer, bookmarksFields, votesContainer, votesFields } from "./common";

export const commentSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchComment'),
    containers: [
        votesContainer(),
        bookmarksContainer(),
        languagesContainer(),
    ],
    fields: [
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesFields(),
    ]
})

export const commentSearchParams = () => toParams(commentSearchSchema(), commentFindMany, CommentSortBy, CommentSortBy.ScoreDesc);