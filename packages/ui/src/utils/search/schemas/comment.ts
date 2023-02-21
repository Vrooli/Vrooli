import { CommentSortBy } from "@shared/consts";
import { commentFindMany } from "api/generated/endpoints/comment";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, bookmarksContainer, bookmarksFields, votesContainer, votesFields } from "./common";

export const commentSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchComment', lng),
    containers: [
        votesContainer(lng),
        bookmarksContainer(lng),
        languagesContainer(lng),
    ],
    fields: [
        ...votesFields(lng),
        ...bookmarksFields(lng),
        ...languagesFields(lng),
    ]
})

export const commentSearchParams = (lng: string) => toParams(commentSearchSchema(lng), commentFindMany, CommentSortBy, CommentSortBy.ScoreDesc);