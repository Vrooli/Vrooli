import { CommentSortBy } from "@shared/consts";
import { commentFindMany } from "api/generated/endpoints/comment";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, starsContainer, starsFields, votesContainer, votesFields } from "./common";

export const commentSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchComment', lng),
    containers: [
        votesContainer,
        starsContainer,
        languagesContainer,
    ],
    fields: [
        ...votesFields,
        ...starsFields,
        ...languagesFields,
    ]
})

export const commentSearchParams = (lng: string) => toParams(commentSearchSchema(lng), commentFindMany, CommentSortBy, CommentSortBy.ScoreDesc);