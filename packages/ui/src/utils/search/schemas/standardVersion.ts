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
        votesRootContainer(lng),
        bookmarksRootContainer(lng),
        languagesContainer(lng),
        tagsRootContainer(lng),
    ],
    fields: [
        ...isCompleteWithRootFields(lng),
        ...isLatestFields(lng),
        ...votesRootFields(lng),
        ...bookmarksRootFields(lng),
        ...languagesFields(lng),
        ...tagsRootFields(lng),
    ]
})

export const standardVersionSearchParams = (lng: string) => toParams(standardVersionSearchSchema(lng), standardVersionFindMany, StandardVersionSortBy, StandardVersionSortBy.DateCreatedDesc);