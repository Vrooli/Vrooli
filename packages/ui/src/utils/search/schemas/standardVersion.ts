import { StandardVersionSortBy } from "@shared/consts";
import { standardVersionFindMany } from "api/generated/endpoints/standardVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const standardVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStandardVersion', lng),
    containers: [
        isCompleteWithRootContainer,
        isLatestContainer,
        votesRootContainer,
        bookmarksRootContainer,
        languagesContainer,
        tagsRootContainer,
    ],
    fields: [
        ...isCompleteWithRootFields,
        ...isLatestFields,
        ...votesRootFields,
        ...bookmarksRootFields,
        ...languagesFields,
        ...tagsRootFields,
    ]
})

export const standardVersionSearchParams = (lng: string) => toParams(standardVersionSearchSchema(lng), standardVersionFindMany, StandardVersionSortBy, StandardVersionSortBy.DateCreatedDesc);