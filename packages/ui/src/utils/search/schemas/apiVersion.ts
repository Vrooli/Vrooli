import { ApiVersionSortBy } from "@shared/consts";
import { apiVersionFindMany } from "api/generated/endpoints/apiVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { isCompleteWithRootContainer, isCompleteWithRootFields, votesRootContainer, votesRootFields, bookmarksRootContainer, bookmarksRootFields, languagesContainer, languagesFields, tagsRootContainer, tagsRootFields, searchFormLayout, isLatestContainer, isLatestFields } from "./common";

export const apiVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchApiVersion', lng),
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

export const apiVersionSearchParams = (lng: string) => toParams(apiVersionSearchSchema(lng), apiVersionFindMany, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc);