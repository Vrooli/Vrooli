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

export const apiVersionSearchParams = (lng: string) => toParams(apiVersionSearchSchema(lng), apiVersionFindMany, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc);